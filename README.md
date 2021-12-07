# waggers
A tool for generating fuzzable URLs from Swagger JSON endpoints. Also contains a basic fuzzer usable with `-dryrun=false`.

Pull requests are welcome!

## Build
```
go get
go build
```

## Run
```
./waggers <options> https://endpoint/swagger.json
```
### Options
Generate URLs (true) or fuzz (false)
```
-dryrun=true|false
```
Output to file
```
-file=urls.txt
```
