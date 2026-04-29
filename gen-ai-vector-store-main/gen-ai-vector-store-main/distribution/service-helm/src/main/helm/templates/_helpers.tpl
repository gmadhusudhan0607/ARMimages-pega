{{- define "provisioningLabels"}}
{{- if (.Values.global) }}{{- if (.Values.global.provisioningTags) }}
{{- $provisioningLabels := .Values.global.provisioningTags }}
{{- range $k,$v := .Values.global.provisioningTags }}
{{ $k }}: {{ regexReplaceAll "[\\/|\\W]+" $v "-"| trunc 63 | trimPrefix "-" | trimSuffix "-" |quote }}
{{- end }}{{- end }}
{{- end }}
{{- end }}