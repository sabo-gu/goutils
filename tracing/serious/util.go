package serious

import (
	opentracing "github.com/opentracing/opentracing-go"
)

const seriousTagKey = "serious"

func SignSerious(sp opentracing.Span, s bool) {
	if s {
		sp.SetTag(seriousTagKey, true)
	} else {
		sp.SetTag(seriousTagKey, false)
	}
}
