package main

import (
	"bytes"
	"strings"
	"text/template"
)

var httpTemplate = `
{{$svrType := .ServiceType}}
{{$svrName := .ServiceName}}

{{- range .MethodSets}}
const Operation{{$svrType}}{{.OriginalName}} = "/{{$svrName}}/{{.OriginalName}}"
{{- end}}

type {{.ServiceType}}HTTPServer interface {
{{- range .MethodSets}}
	{{- if ne .Comment ""}}
	{{.Comment}}
	{{- end}}
	{{.Name}}(context.Context, *{{.Request}}) (*{{.Reply}}, error)
{{- end}}
}

func Register{{.ServiceType}}HTTPServer(s *gin.Engine, srv {{.ServiceType}}HTTPServer) {
	r := s.Group("/")
	{{- range .Methods}}
	r.{{.Method}}("{{.Path}}", _{{$svrType}}_{{.Name}}{{.Num}}_HTTP_Handler(srv))
	{{- end}}
}

{{range .Methods}}
func _{{$svrType}}_{{.Name}}{{.Num}}_HTTP_Handler(srv {{$svrType}}HTTPServer) gin.HandlerFunc {
	return func(c *gin.Context) {
		transformer := message.GetTransformer(message.DefaultTransformerName)

		var in {{.Request}}
		if err := c.ShouldBind(&in);err != nil {
			statusCode, data, e := transformer.Err(c.Request.Context(), err)
			if e != nil {
				c.JSON(statusCode, e.Error())
				c.Abort()

				return
			}

			c.Data(statusCode, transformer.ContentType(), data)
			c.Abort()

			return
		}

		if err := in.Validate();err != nil {
			statusCode, data, e := transformer.Err(c.Request.Context(), validate.NewValidateError(err))
			if e != nil {
				c.JSON(statusCode, e.Error())
				c.Abort()

				return
			}

			c.Data(statusCode, transformer.ContentType(), data)
			c.Abort()

			return
		}

		out, err := srv.{{.Name}}(c.Request.Context(), &in)
		if err != nil {
			statusCode, data, e := transformer.Err(c.Request.Context(), err)
			if e != nil {
				c.JSON(statusCode, e.Error())
				c.Abort()

				return
			}

			c.Data(statusCode, transformer.ContentType(), data)
			c.Abort()

			return
		}

		httpStatusCode, data, err := transformer.Transform(c.Request.Context(), out)
		if err != nil {
			statusCode, data, e := transformer.Err(c.Request.Context(), err)
			if e != nil {
				c.JSON(statusCode, e.Error())
				c.Abort()

				return
			}

			c.Data(statusCode, transformer.ContentType(), data)
			c.Abort()

			return
		}

		c.Data(httpStatusCode, transformer.ContentType(), data)
		c.Abort()
	}
}
{{end}}
`

type serviceDesc struct {
	ServiceType string // Greeter
	ServiceName string // helloworld.Greeter
	Metadata    string // api/helloworld/helloworld.proto
	Methods     []*methodDesc
	MethodSets  map[string]*methodDesc
}

type methodDesc struct {
	// method
	Name         string
	OriginalName string // The parsed original name
	Num          int
	Request      string
	Reply        string
	Comment      string
	// http_rule
	Path         string
	Method       string
	HasVars      bool
	HasBody      bool
	Body         string
	ResponseBody string
}

func (s *serviceDesc) execute() string {
	s.MethodSets = make(map[string]*methodDesc)
	for _, m := range s.Methods {
		s.MethodSets[m.Name] = m
	}
	buf := new(bytes.Buffer)
	tmpl, err := template.New("http").Parse(strings.TrimSpace(httpTemplate))
	if err != nil {
		panic(err)
	}
	if err := tmpl.Execute(buf, s); err != nil {
		panic(err)
	}
	return strings.Trim(buf.String(), "\r\n")
}
