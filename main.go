package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
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
	fmt.Print(banner)
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
		panic("JSON parsing error " + unmarshalErr.Error())
	}

	return &swaggerResp
}

func usage() {
	fmt.Println("waggers OPTIONS <url>")
	flag.PrintDefaults()
}

func buildFuzzableUrl(api *swagger.SwaggerApiProps, scheme string, hostname string) (string, error) {
	var fullUrl string
	for i := 0; i < 10; i++ {
		tempUrl := hostname + buildApiPath(api)
		if !strings.HasPrefix(tempUrl, scheme) {
			tempUrl = scheme + tempUrl
		}
		_, parseErr := url.Parse(tempUrl)
		if parseErr != nil {
			continue
		}
		fullUrl = tempUrl
		break
	}
	if len(fullUrl) > 0 {
		return fullUrl, nil
	}
	return "", errors.New("Couldn't build fuzzable url " + fullUrl)
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

func fuzz(url string, httpClient *http.Client, fuzzChannel chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	fuzzResp, fuzzErr := httpClient.Get(url)
	if fuzzResp != nil {
		fuzzChannel <- "[" + strconv.Itoa(fuzzResp.StatusCode) + "] " + url + "\n"
	} else {
		if fuzzErr != nil {
			fuzzChannel <- "Fuzzer error " + fuzzErr.Error() + " - " + url + "\n"
		}
	}
}

func getHostName(swaggerResp swagger.SwaggerResponse) string {
	if swaggerResp.OpenApi != nil {
		if swaggerResp.Servers == nil || len(swaggerResp.Servers) == 0 {
			fmt.Println("Null or empty array in OpenAPI definition")
			os.Exit(1)
		}
		return *&swaggerResp.Servers[0].Url
	}
	return *swaggerResp.Host
}

func main() {
	printBanner()
	rand.Seed(time.Now().UnixNano())

	dryRun := flag.Bool("dryrun", true, "Only print URLs, no fuzzing")
	fuzzCount := flag.Int("fuzzcount", 1, "How many fuzzable URLs should be generated/fuzzed, the default is 1")
	outFile := flag.String("file", "", "Output file")
	flag.Parse()
	flag.Usage = usage

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	if *fuzzCount <= 0 {
		fmt.Println("Fuzz count must be greater than 0")
		os.Exit(1)
	} else if *fuzzCount > 1000 {
		fmt.Println("Limit of fuzzable URLs is 1000")
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

	var wg sync.WaitGroup
	start := time.Now()

	scheme := url.Scheme + "://"
	hostname := getHostName(*swaggerResp)

	for _, api := range parsed.Paths {
		if len(api.Params) == 0 {
			continue
		}
		fuzzChannel := make(chan string)

		for i := 0; i < *fuzzCount; i++ {
			fullUrl, fuzzableUrlErr := buildFuzzableUrl(&api, scheme, hostname)
			if fuzzableUrlErr != nil {
				fmt.Println("Couldn't build fuzzable URL for " + api.Path)
			}
			if *dryRun {
				writer.WriteString(fullUrl + "\n")
				writer.Flush()
			} else {
				wg.Add(1)
				go fuzz(fullUrl, httpClient, fuzzChannel, &wg)
				fuzzMsg := <-fuzzChannel
				writer.WriteString(fuzzMsg)
			}
		}
	}

	wg.Wait()

	if !*dryRun {
		fmt.Printf("Fuzzing took %s", time.Since(start))
	}
}
