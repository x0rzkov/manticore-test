package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"bytes"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/nozzle/throttler"
	"github.com/manticoresoftware/go-sdk/manticore"
	"github.com/spf13/pflag"
)

var (
	manticoreHost string
	manticorePort uint16
	inputFile     string
	parallelJobs int
	help          bool
)

func main() {
	pflag.StringVarP(&manticoreHost, "manticore-host", "m", "localhost", "input-file")
	pflag.Uint16VarP(&manticorePort, "manticore-port", "p", 9312, "input-file")
	pflag.IntVarP(&parallelJobs, "parallel-jobs", "j", 2, "parallel-jobs")
	pflag.StringVarP(&inputFile, "input-file", "f", "/opt/manticore/data/manticore-dump.sql", "input-file")
	pflag.BoolVarP(&help, "help", "h", false, "display help")
	pflag.Parse()
	if help {
		pflag.PrintDefaults()
		os.Exit(1)
	}

	cl, _, err := initSphinx(manticoreHost, manticorePort)
	check(err)

	file, err := os.Open(inputFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	lineCount, err := lineCounter(inputFile)
	if err != nil {
		log.Fatal(err)
	}

	t := throttler.New(parallelJobs, lineCount)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {

		go func() error {
			// Let Throttler know when the goroutine completes
			// so it can dispatch another worker
			defer t.Done(nil)
			query := scanner.Text()
			log.Info("query:", query)
			resp, err := cl.Sphinxql(query)
			if err != nil {
				log.Println("query error: ", query)
				log.Fatalln(err)
				return fmt.Errorf("%s", err)
			}
			if resp[0].Msg != "" {
				log.Println("query msg: ", query)
				log.Fatalln(resp[0].Msg)
				return fmt.Errorf("%s", resp[0].Msg)
			}
			return nil
		}()

		t.Throttle()
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	// throttler errors iteration
	if t.Err() != nil {
		// Loop through the errors to see the details
		for i, err := range t.Errs() {
			log.Printf("error #%d: %s", i, err)
		}
		log.Fatal(t.Err())
	}	

}

func initSphinx(host string, port uint16) (manticore.Client, bool, error) {
	cl := manticore.NewClient()
	cl.SetServer(host, port)
	status, err := cl.Open()
	if err != nil {
		return cl, status, err
	}
	return cl, status, nil
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// Untar takes a destination path and a reader; a tar reader loops over the tarfile
// creating the file structure at 'dst' along the way, and writing any files
func untar(dst string, r io.Reader) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			return nil

		// return any other error
		case err != nil:
			return err

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		// the target location where the dir/file should be created
		target := filepath.Join(dst, header.Name)

		// the following switch could also be done using fi.Mode(), not sure if there
		// a benefit of using one vs. the other.
		// fi := header.FileInfo()

		// check the file type
		switch header.Typeflag {
		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}
		// if it's a file create it
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}
			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			f.Close()
		}
	}
}

func lineCounter(inputFile string) (int, error) {
	file, err := os.Open(inputFile)
	if err != nil {
		log.Fatal(fmt.Sprintf("Couldn't open the csv file: \n'%s'\n", err))
	}

	var readSize int
	// var err error
	var count int

	buf := make([]byte, 1024)

	for {
		readSize, err = file.Read(buf)
		if err != nil {
			break
		}

		var buffPosition int
		for {
			i := bytes.IndexByte(buf[buffPosition:], '\n')
			if i == -1 || readSize == buffPosition {
				break
			}
			buffPosition += i + 1
			count++
		}
	}
	if readSize > 0 && count == 0 || count > 0 {
		count++
	}
	if err == io.EOF {
		return count, nil
	}

	return count, err
}
