package swagger

type ApiProps struct {
	Tags        []string `json:"tags"`
	Summary     string   `json:"summary"`
	OperationID string   `json:"operationId"`
	Produces    []string `json:"produces"`
	Parameters  []struct {
		Name        string `json:"name"`
		In          string `json:"in"`
		Description string `json:"description"`
		Required    bool   `json:"required"`
		Type        string `json:"type"`
		Format      string `json:"format"`
	} `json:"parameters"`
	Responses struct {
		Num200 struct {
			Description string `json:"description"`
			Schema      struct {
				Ref string `json:"$ref"`
			} `json:"schema"`
		} `json:"200"`
	} `json:"responses"`
}

type SwaggerResponse struct {
	Swagger string `json:"swagger"`
	Info    struct {
		Description    string `json:"description"`
		Version        string `json:"version"`
		Title          string `json:"title"`
		TermsOfService string `json:"termsOfService"`
		Contact        struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"contact"`
		License struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"license"`
	} `json:"info"`
	Host     string `json:"host"`
	BasePath string `json:"basePath"`
	Tags     []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"tags"`
	Produces []string `json:"produces"`
	Paths    map[string]struct {
		Get ApiProps
	} `json:"paths"`
	Definitions         map[string]interface{} `json:"definitions"`
	SecurityDefinitions interface{}            `json:"securityDefinitions"`
}
