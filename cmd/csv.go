package cmd

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/spf13/cobra"
)

var (
	srcPath string = ""
	dstPath string = "./"
	nworker        = 10
)

type file struct {
	URL    string
	SaveTo string
}

func init() {
	csvCmd.PersistentFlags().StringVar(&srcPath, "s", "", "Source file path")
	csvCmd.PersistentFlags().StringVar(&dstPath, "d", "./", "Destitnation folder path")
	csvCmd.PersistentFlags().IntVar(&nworker, "w", 10, "Number of workers")

	rootCmd.AddCommand(csvCmd)
}

var csvCmd = &cobra.Command{
	Use:   "csv",
	Short: "Download the files using the csv",
	Long: `Format of the should be three columns 
Folder Name | File Name | File URL
ABC	     | 4445d3a	| https://example.com/img.jpg
	`,
	Run: func(cmd *cobra.Command, args []string) {
		defer func(since time.Time) {
			log.Printf("time tooks: %v\n", time.Since(since))
			log.Println("Finished.")
		}(time.Now())

		if srcPath == "" {
			log.Fatal("missing source csv file")
		}

		log.Println("Starting downloading...")
		records := readCsvFile(srcPath)
		records = records[1:]
		bar := pb.StartNew(len(records))

		ctx, _ := context.WithCancel(context.Background())
		workerCh := make(chan file, 10)
		wg := &sync.WaitGroup{}
		startWorkers(wg, ctx, workerCh, bar, nworker)

		for i, record := range records {

			if len(record) < 3 {
				log.Fatal(fmt.Errorf("mismatched column at line %d", i+2))
			}

			_, err := url.ParseRequestURI(record[2])
			if err != nil {
				log.Fatal(fmt.Errorf("invalid url at line %d, err msg = %v", i+2, err))
			}

			if err := createDirIfNeeded(record[0]); err != nil {
				log.Fatal(err)
			}

			workerCh <- file{
				URL:    record[2],
				SaveTo: fmt.Sprintf("%s/%s", record[0], record[1]),
			}
		}

		close(workerCh)
		wg.Wait()
		bar.Finish()

	},
}

func createDirIfNeeded(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.Mkdir(path, 0755)
		if err != nil {
			return err
		}
	}

	return nil
}

func startWorkers(wg *sync.WaitGroup, ctx context.Context, ch chan file, bar *pb.ProgressBar, nworkers int) {

	i := 0
	for i < nworkers {
		wg.Add(1)
		go worker(wg, ch, bar)
		i++
	}

}

func worker(wg *sync.WaitGroup, ch <-chan file, bar *pb.ProgressBar) {
	defer wg.Done()

	for msg := range ch {
		// log.Printf("downloading: %s\n", msg.URL)
		if err := download(msg.URL, msg.SaveTo); err != nil {
			log.Fatal(err)
		}
		bar.Increment()
		time.Sleep(100 * time.Millisecond)
	}
}

func download(src, dst string) error {

	response, err := http.Get(src)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return errors.New("received non 200 response code")
	}

	file, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return err
	}

	return nil
}

func readCsvFile(filePath string) [][]string {
	f, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("unable to read input file %s, err msg= %s", filePath, err.Error())
	}
	defer f.Close()

	csvReader := csv.NewReader(f)
	records, err := csvReader.ReadAll()
	if err != nil {
		log.Fatalf("unable to parse file as csv for %s, err msg= %s", filePath, err.Error())
	}

	return records
}
