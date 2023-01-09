package pas2go

import "text/template"

var (
	tmplFuncs = template.FuncMap{
		"toGoType":                 toGoType,
		"sortTypeDefs":             sortTypeDefs,
		"constantLiteral":          constantLiteral,
		"constantLiteralList":      constantLiteralList,
		"formalParams":             formalParams,
		"actualParams":             actualParams,
		"toExpr":                   toExpr,
		"filterEnumTypes":          filterEnumTypes,
		"generateEnumConstants":    generateEnumConstants,
		"isBuiltinProcedure":       isBuiltinProcedure,
		"generateBuiltinProcedure": generateBuiltinProcedure,
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
	{{- template "typeConstants" .Types }}
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

{{- define "typeConstants" }}
	{{- $types := . | filterEnumTypes }}
	{{- if $types }}
		{{- range $type := $types }}
			{{- $type | generateEnumConstants }}
		{{- end }}
	{{- end -}}
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
		var {{ $routine.Name }} func({{ $routine.FormalParameters | formalParams }}){{ if $routine.ReturnType }} {{ $routine.ReturnType | toGoType }}{{ end }}
		{{ $routine.Name }} = func({{ $routine.FormalParameters | formalParams }}){{ if $routine.ReturnType }} ({{ $routine.Name }}_ {{ $routine.ReturnType | toGoType }}){{ end }} {
			{{- template "block" $routine.Block }}
			return
		}
	{{ end -}}
{{ end }}

{{- define "statements" }}
	{{- range $statement := . }}
		{{- if $statement.Label }}
		L{{ $statement.Label }}:
		{{- end }}
		{{- template "statement" $statement }}
	{{- end -}}
{{ end }}


{{- define "statement" }}
	{{- if eq .Type 0 }}{{/* goto */}}
		goto L{{ .Target }}
	{{- else if eq .Type 1 }}{{/* assignment */}}
		{{ template "expr" .LeftExpr }} = {{ template "expr" .RightExpr }}
	{{- else if eq .Type 2 }}{{/* procedure call */}}
		{{ if isBuiltinProcedure .Name -}}
			{{ generateBuiltinProcedure . }}
		{{- else -}}
			{{ .Name }}{{  actualParams .ActualParams .FormalParams }}
		{{- end }}
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
			{{- template "statement" .Statement }}
		}
		{{- if .ElseStatement }} else {
			{{- template "statement" .ElseStatement }}
		}
		{{- end }}
	{{- else if eq .Type 8 }}{{/* case statement */}}
		switch {{ template "expr" .Expr }} {
		{{- range $caseLimb := .CaseLimbs }}
		case {{ $caseLimb.Label | constantLiteralList }}:
			{{- template "statement" $caseLimb.Statement }}
		{{- end }}
		}
	{{- else if eq .Type 9 }}{{/* with statement */}}
		{{ template "statements" .Block.Statements }}
	{{- else if eq .Type 10 }}{{/* write statement */}}
		system.Write{{ if .AppendNewLine }}ln{{ end }}{{ actualParams .ActualParams nil }}
	{{- else }}
	// bug: invalid statement type {{ .Type }}
	{{- end }}
{{- end }}

{{- define "expr" }}
{{- . | toExpr }}
{{- end }}
`
