package swagger

import (
	"strings"

	fuzz "github.com/google/gofuzz"
)

var fuzzer = fuzz.New()

type SwaggerApiParamType int64

const (
	String SwaggerApiParamType = iota
	Integer
	Float
)

type SwaggerApiParam struct {
	Name           string
	Type           SwaggerApiParamType
	Required       bool
	IsPathVariable bool
}

func (p *SwaggerApiParam) Fuzz() interface{} {
	var fuzzedValue interface{}
	switch p.Type {
	case Integer:
		{
			var paramInt int64
			fuzzer.Fuzz(&paramInt)
			fuzzedValue = paramInt
		}
	case Float:
		{
			var paramNum float64
			fuzzer.Fuzz(&paramNum)
			fuzzedValue = paramNum
		}
	default:
		{
			var paramStr string
			fuzzer.Fuzz(&paramStr)
			fuzzedValue = paramStr
		}
	}
	return fuzzedValue
}

type SwaggerApiProps struct {
	Path   string
	Params []SwaggerApiParam
}

type SwaggerApi struct {
	Paths []SwaggerApiProps
}

func ParseSwagger(resp *SwaggerResponse) SwaggerApi {
	var api = SwaggerApi{}
	api.Paths = make([]SwaggerApiProps, 0)
	for path, props := range resp.Paths {
		params := make([]SwaggerApiParam, 0)
		for _, param := range props.Get.Parameters {
			var paramType SwaggerApiParamType
			if param.Type == "number" {
				paramType = Float
			} else if param.Type == "integer" {
				paramType = Integer
			} else {
				paramType = String
			}
			params = append(params, SwaggerApiParam{
				Name:           param.Name,
				Type:           paramType,
				Required:       param.Required,
				IsPathVariable: strings.Contains(path, "{"+param.Name+"}"),
			})
		}
		api.Paths = append(api.Paths, SwaggerApiProps{
			Path:   path,
			Params: params,
		})
	}
	return api
}
