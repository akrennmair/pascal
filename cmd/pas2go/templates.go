package main

const sourceTemplate = `
{{- define "main" -}}
package main

/*
import (
	"github.com/akrennmair/pascal/pas2go/system"
)
*/

func main() {
	// program {{ .Name }}
	{{- template "block" .Block }}

}
{{ end }}
{{- define "block" }}
	{{- template "constants" .Constants }}

	{{ template "types" .Types }}

	{{ template "variables" .Variables }}

	{{ template "functions" .Procedures }}

	{{ template "functions" .Functions }}

	{{ template "statements" .Statements }}
{{- end }}

{{- define "constants" }}
	{{- if . }}
	const (
	{{- range $const := . }}
		{{ $const.Name }} = {{ $const.Value | constantLiteral }}
	{{- end }}
	)
	{{- end }}
{{- end }}

{{- define "types" }}
	{{- range $type := . }}
	type {{ $type.Name }} {{ $type.Type | toGoType }}
	{{ end }}
{{- end }}

{{- define "variables" }}
	{{- if . }}
	var (
	{{- range $var := . }}
		{{ $var.Name }} {{ $var.Type | toGoType }}
	{{- end }}
	)
	{{- end }}
{{- end }}

{{- define "functions" }}
	{{- range $routine := . }}
		{{ $routine.Name }} := func({{ $routine.FormalParameters | formalParams }}){{ if $routine.ReturnType }} {{ $routine.ReturnType | toGoType }}{{ end }} {
			{{ template "block" $routine.Block }}
		}
	{{- end }}
{{- end }}

{{- define "statements" }}
	{{- range $statement := . }}
		{{- if $statement.Label }}
		{{ $statement.Label }}:
		{{- end }}
		{{ template "statement" $statement }}
	{{- end }}
{{- end }}


{{- define "statement" }}
	{{- if eq .Type 0 }}{{/* goto */}}
		goto {{ .Target }}
	{{- else if eq .Type 1 }}{{/* assignment */}}
		{{ template "expr" .LeftExpr }} = {{ template "expr" .RightExpr }}
	{{- else if eq .Type 2 }}{{/* procedure call */}}
		// TODO: implement procedure call
	{{- else if eq .Type 3 }}{{/* compound statement */}}
		{
			{{ template "statements" .Statements }}
		}
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
		for {{ .Name }} = {{ template "expr" .InitialExpr }}; {{ .Name }} <= {{ template "expr" .FinalExpr }}; {{ .Name }}{{ if .DownTo }}--{{ else }}++{{ end }} {
			{{ template "statement" .Statement }}
		}
	{{- else if eq .Type 7 }}{{/* if statement */}}
		if {{ template "expr" .Condition }} {
			{{ template "statement" .Statement }}
		}
	{{- else if eq .Type 8 }}{{/* case statement */}}
		// TODO: implement case
	{{- else if eq .Type 9 }}{{/* with statement */}}
		// TODO: implement with
	{{- else if eq .Type 10 }}{{/* write statement */}}
		// TODO: implement write
	{{- else }}
	// bug: invalid statement type {{ .Type }}
	{{- end}}
{{- end }}

{{- define "expr" }}
{{- . | toExpr }}
{{- end }}
`
