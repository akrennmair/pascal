package pas2go

import "text/template"

var (
	tmplFuncs = template.FuncMap{
		"toGoType":        toGoType,
		"sortTypeDefs":    sortTypeDefs,
		"constantLiteral": constantLiteral,
		"formalParams":    formalParams,
		"actualParams":    actualParams,
		"toExpr":          toExpr,
	}
	transpilerTemplate = template.Must(template.New("").Funcs(tmplFuncs).Parse(sourceTemplate))
)

const sourceTemplate = `
{{- define "main" -}}
package main

import (
	"github.com/akrennmair/pascal/pas2go/system"
)

var _ = system.Write

// program {{ .Name }}
func main() {
	{{- template "block" .Block }}
}
{{ end }}

{{- define "block" }}
	{{- template "constants" .Constants }}
	{{- template "types" .Types }}
	{{- template "variables" .Variables }}
	{{- template "functions" .Procedures }}
	{{- template "functions" .Functions }}
	{{- template "statements" .Statements }}
{{- end }}

{{- define "constants" }}
	{{- if . }}
	const (
	{{- range $const := . }}
		{{ $const.Name }} = {{ $const.Value | constantLiteral }}
	{{- end }}
	)
	{{ end -}}
{{ end }}

{{- define "types" }}
	{{- if . }}
	type (
	{{- range $type := . | sortTypeDefs }}
	{{ $type.Name }} {{ $type.Type | toGoType }}
	{{- end }}
	)
	{{ end -}}
{{ end }}

{{- define "variables" }}
	{{- if . }}
	var (
	{{- range $var := . }}
		{{ $var.Name }} {{ $var.Type | toGoType }}
	{{- end }}
	)

	{{- range $var := . }}
	_ = {{ $var.Name }}
	{{- end }}
	{{ end -}}
{{ end }}

{{- define "functions" }}
	{{- range $routine := . }}
		{{ $routine.Name }} := func({{ $routine.FormalParameters | formalParams }}){{ if $routine.ReturnType }} ({{ $routine.Name }} {{ $routine.ReturnType | toGoType }}){{ end }} {
			{{- template "block" $routine.Block }}
			return
		}
	{{ end -}}
{{ end }}

{{- define "statements" }}
	{{- range $statement := . }}
		{{- if $statement.Label }}
		{{ $statement.Label }}:
		{{- end }}
		{{- template "statement" $statement }}
	{{- end -}}
{{ end }}


{{- define "statement" }}
	{{- if eq .Type 0 }}{{/* goto */}}
		goto {{ .Target }}
	{{- else if eq .Type 1 }}{{/* assignment */}}
		{{ template "expr" .LeftExpr }} = {{ template "expr" .RightExpr }}
	{{- else if eq .Type 2 }}{{/* procedure call */}}
		{{ .Name }}{{ .ActualParams | actualParams }}
	{{- else if eq .Type 3 }}{{/* compound statement */}}
		{{- template "statements" .Statements }}
	{{- else if eq .Type 4 }}{{/* while statement */}}
		for {{ template "expr" .Condition }} {
			{{ template "statement" .Statement }}
		}
	{{- else if eq .Type 5 }}{{/* repeat statement */}}
		for {
			{{ template "statements" .Statements }}

			if {{ template "expr" .Condition }} {
				break
			}
		}
	{{- else if eq .Type 6 }}{{/* for statement */}}
		for {{ .Name }} = {{ template "expr" .InitialExpr }}; {{ .Name }} {{ if .DownTo }}>={{ else }}<={{ end }} {{ template "expr" .FinalExpr }}; {{ .Name }}{{ if .DownTo }}--{{ else }}++{{ end }} {
			{{- template "statement" .Statement }}
		}
	{{- else if eq .Type 7 }}{{/* if statement */}}
		if {{ template "expr" .Condition }} {
			{{ template "statement" .Statement }}
		}
	{{- else if eq .Type 8 }}{{/* case statement */}}
		// TODO: implement case
	{{- else if eq .Type 9 }}{{/* with statement */}}
		{{ template "statements" .Statement.Block.Statements }}
	{{- else if eq .Type 10 }}{{/* write statement */}}
		system.Write{{ if .AppendNewLine }}ln{{ end }}{{ .ActualParams | actualParams }}
	{{- else }}
	// bug: invalid statement type {{ .Type }}
	{{- end }}
{{- end }}

{{- define "expr" }}
{{- . | toExpr }}
{{- end }}
`
