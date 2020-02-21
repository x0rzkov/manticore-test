package main

import (
    "bufio"
    "fmt"
    "log"
    "os"

	"github.com/spf13/pflag"
	"github.com/x0rzkov/go-sdk/manticore"
)

var (
	manticoreHost string
	manticorePort string
	inputFile string
)

func main() {
	pflag.StringVarP(&manticoreHost, "manticore-host", "h", "localhost", "input-file")
	pflag.IntVarP(&manticorePort, "manticore-port", "p", 9312, "input-file")
	pflag.StringVarP(&inputFile, "input-file", "f", "./manticore.sql", "input-file")
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

    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
		resp, err := cl.Sphinxql(scanner.Text())
		if err != nil {
			log.Println(scanner.Text())
			log.Fatalln(err)
		}
		if resp[0].Msg != "" {
			log.Println(scanner.Text())
			log.Fatalln(resp[0].Msg)
		}
    }

    if err := scanner.Err(); err != nil {
        log.Fatal(err)
    }

}

func initSphinx(host string, port uint16) (manticore.Client, bool, error) {
	cl := manticore.NewClient()
	cl.SetServer("localhost", 9312)
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