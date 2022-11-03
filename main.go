package main

import (
	"bufio"
	"crypto/tls"
	base64 "encoding/base64"
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

func buildFuzzableUrl(api *swagger.SwaggerApiProps, scheme string, hostname string, fuzzword string) (string, error) {
	var fullUrl string
	for i := 0; i < 10; i++ {
		tempUrl := hostname + buildApiPath(api, fuzzword)
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

func buildApiPath(api *swagger.SwaggerApiProps, fuzzword string) string {
	ret := api.Path + "?"
	for i, param := range api.Params {
		var actualFuzzWord interface{}
		if len(fuzzword) == 0 {
			actualFuzzWord = param.Fuzz()
		} else {
			actualFuzzWord = fuzzword
		}
		if param.IsPathVariable {
			ret = strings.Replace(ret, "{"+param.Name+"}", fmt.Sprintf("%v", actualFuzzWord), 1)
		} else {
			ret += param.Name + "=" + fmt.Sprintf("%v", actualFuzzWord)
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

func fuzz(url string, httpHeaders []string, httpClient *http.Client, fuzzChannel chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	req, reqErr := http.NewRequest("GET", url, nil)
	for _, header := range httpHeaders {
		headerSplit := strings.Split(header, ":")
		req.Header.Add(headerSplit[0], headerSplit[1])
	}
	if reqErr != nil {
		fuzzChannel <- "HTTP client error " + reqErr.Error() + " - " + url + "\n"
	}
	fuzzResp, fuzzErr := httpClient.Do(req)
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
		return swaggerResp.Servers[0].Url
	}
	return *swaggerResp.Host
}

func main() {
	printBanner()
	rand.Seed(time.Now().UnixNano())

	var customHeaders []string

	dryRun := flag.Bool("dryrun", true, "Only print URLs, no fuzzing")
	fuzzCount := flag.Int("fuzzcount", 1, "How many fuzzable URLs should be generated/fuzzed, the default is 1")
	fuzzWord := flag.String("fuzzword", "", "A custom fuzz word (e.g. FUZZ) inserted into URLs. Using a fuzz word ignores the fuzz count flag.")
	customHeadersArg := flag.String("headers", "", "Custom HTTP headers separated with a comma, e.g. 'Content-Type: application/json,User-Agent:foobar.'")
	outFile := flag.String("file", "", "Output file")
	shuffle := flag.Bool("shuffle", false, "Shuffle URL list")
	ignoreCertErrors := flag.Bool("ignorecert", false, "Ignore certificate errors")
	basicAuth := flag.String("basicauth", "", "Convenience method for basic authentication, e.g. user:pass")

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

	if len(*customHeadersArg) > 0 {
		customHeaders = strings.Split(*customHeadersArg, ",")
	}

	if *ignoreCertErrors {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	if len(*basicAuth) > 0 {
		encodedAuth := base64.StdEncoding.EncodeToString([]byte(*basicAuth))
		customHeaders = append(customHeaders, "Authorization:Basic "+encodedAuth)
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
	paths := parsed.Paths

	if *shuffle {
		rand.Shuffle(len(paths), func(i, j int) { paths[i], paths[j] = paths[j], paths[i] })
	}

	if *dryRun {
		fmt.Println("Dry run, just printing URLs. Use -dryrun=false to fuzz.")
		fmt.Println()
	}

	var wg sync.WaitGroup
	start := time.Now()

	scheme := url.Scheme + "://"
	hostname := getHostName(*swaggerResp)

	for _, api := range paths {
		api := api

		if len(api.Params) == 0 {
			continue
		}
		fuzzChannel := make(chan string)

		for i := 0; i < *fuzzCount; i++ {
			fullUrl, fuzzableUrlErr := buildFuzzableUrl(&api, scheme, hostname, *fuzzWord)
			if fuzzableUrlErr != nil {
				fmt.Println("Couldn't build fuzzable URL for " + api.Path)
			}
			if *dryRun {
				writer.WriteString(fullUrl + "\n")
				writer.Flush()
			} else {
				wg.Add(1)
				go fuzz(fullUrl, customHeaders, httpClient, fuzzChannel, &wg)
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
