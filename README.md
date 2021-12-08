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

### Output
```
./waggers https://petstore.swagger.io/v2/swagger.json

https://petstore.swagger.io/user/login?username=T顬'ų共ưć埂Û蠖ExV囅&password=;4鉜*J7
https://petstore.swagger.io/user/şmȭGƮ¨Ɓ卵ǥ
https://petstore.swagger.io/pet/findByTags?tags=Ć4癶$
https://petstore.swagger.io/pet/-582776452570529845
https://petstore.swagger.io/store/order/2367787863121478808
https://petstore.swagger.io/pet/findByStatus?status=暐!ü嚀渣ƙlȬ秡韫镰qʋ

```