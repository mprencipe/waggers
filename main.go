package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"waggers/internal/swagger"
)

const banner = `
\ \/ \/ /\__  \   / ___\ / ___\_/ __ \_  __ \/  ___/
 \     /  / __ \_/ /_/  > /_/  >  ___/|  | \/\___ \ 
  \/\_/  (____  /\___  /\___  / \___  >__|  /____  >
              \//_____//_____/      \/           \/                                                   
`

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

func fuzz(url string, httpClient *http.Client) {
	// nothing yet
}

func main() {
	printBanner()

	dryRun := flag.Bool("dryrun", true, "Only print URLs, no fuzzing")
	outFile := flag.String("outfile", "", "Output file")
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

	var output *os.File
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
			fuzz(fullUrl, httpClient)
		}
	}
}
