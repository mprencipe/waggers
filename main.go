package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	"waggers/internal/swagger"
)

const banner = `
\ \/ \/ /\__  \   / ___\ / ___\_/ __ \_  __ \/  ___/
 \     /  / __ \_/ /_/  > /_/  >  ___/|  | \/\___ \ 
  \/\_/  (____  /\___  /\___  / \___  >__|  /____  >
              \//_____//_____/      \/           \/                                                   
`

var output *os.File

func printBanner() {
	fmt.Println(banner)
}

func getSwaggerResponse(url string, httpClient *http.Client) *swagger.SwaggerResponse {
	resp, err := httpClient.Get(url)
	if err != nil {
		panic(err)
	}

	var swaggerResp swagger.SwaggerResponse

	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		panic(readErr)
	}

	unmarshalErr := json.Unmarshal(body, &swaggerResp)
	if unmarshalErr != nil {
		panic(unmarshalErr)
	}

	return &swaggerResp
}

func usage() {
	fmt.Println("waggers OPTIONS <url>")
	flag.PrintDefaults()
}

func buildApiPath(api *swagger.SwaggerApiProps) string {
	ret := api.Path + "?"
	for i, param := range api.Params {
		if param.IsPathVariable {
			ret = strings.Replace(ret, "{"+param.Name+"}", fmt.Sprintf("%v", param.Fuzz()), 1)
		} else {
			ret += param.Name + "=" + fmt.Sprintf("%v", param.Fuzz())
			if i != len(api.Params)-1 {
				ret += "&"
			}
		}
	}
	if strings.HasSuffix(ret, "?") {
		ret = ret[:len(ret)-1]
	}
	return ret
}

func fuzz(url string, httpClient *http.Client, writer *bufio.Writer) {
	fuzzResp, fuzzErr := httpClient.Get(url)
	if fuzzResp != nil {
		writer.WriteString("[" + strconv.Itoa(fuzzResp.StatusCode) + "] " + url + "\n")
	} else {
		if fuzzErr != nil {
			writer.WriteString("Fuzzer error " + fuzzErr.Error() + " - " + url + "\n")
		}
	}
}

func main() {
	printBanner()
	rand.Seed(time.Now().UnixNano()) // seed random generator

	dryRun := flag.Bool("dryrun", true, "Only print URLs, no fuzzing")
	outFile := flag.String("file", "", "Output file")
	flag.Parse()
	flag.Usage = usage

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	urlArg := flag.Arg(0)
	url, urlErr := url.ParseRequestURI(urlArg)
	if urlErr != nil {
		panic(urlErr)
	}

	var openErr error
	if len(*outFile) > 0 {
		output, openErr = os.OpenFile(*outFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if openErr != nil {
			panic(openErr)
		}
	} else {
		output = os.Stdout
	}
	writer := bufio.NewWriter(output)

	httpClient := &http.Client{}
	swaggerResp := getSwaggerResponse(urlArg, httpClient)
	parsed := swagger.ParseSwagger(swaggerResp)

	if *dryRun {
		fmt.Println("Dry run, just printing URLs. Use -dryrun=false to fuzz.")
		fmt.Println()
	}

	for _, api := range parsed.Paths {
		if len(api.Params) == 0 {
			continue
		}

		fullUrl := url.Scheme + "://" + swaggerResp.Host + buildApiPath(&api)

		if *dryRun {
			writer.WriteString(fullUrl + "\n")
			writer.Flush()
		} else {
			fuzz(fullUrl, httpClient, writer)
			writer.Flush()
		}
	}
}
