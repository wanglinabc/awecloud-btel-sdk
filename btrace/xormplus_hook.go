package btrace

import (
	"context"
	"github.com/xormplus/xorm"
	"github.com/xormplus/xorm/contexts"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type xormplusHook struct {
	tracer oteltrace.Tracer
	engine *xorm.Engine
}

func NewXormplusHook(engine *xorm.Engine, tracer oteltrace.Tracer) *xormplusHook {
	return &xormplusHook{tracer: tracer, engine: engine}
}

func XormplusWrapEngine(e *xorm.Engine, tracer oteltrace.Tracer) {
	e.AddHook(NewXormplusHook(e, tracer))
}

func (h *xormplusHook) BeforeProcess(c *contexts.ContextHook) (context.Context, error) {
	connect, user, dbName := connParse(string(h.engine.Dialect().URI().DBType), h.engine.DataSourceName())
	commonAttrs := []attribute.KeyValue{
		attribute.String("db.statement", c.SQL),
		attribute.String("db.connection_string", connect),
		attribute.String("db.name", dbName),
		attribute.String("db.system", string(h.engine.Dialect().URI().DBType)),
		attribute.String("db.user", user),
		attribute.String("db.operation", getSqlOperation(c.SQL)),
		attribute.String("db.sql.table", getTableName(c.SQL)),
	}
	_, iSpan := h.tracer.Start(c.Ctx, getSqlOperation(c.SQL)+" "+h.engine.Dialect().URI().DBName, trace.WithAttributes(commonAttrs...))
	ctx := context.WithValue(c.Ctx, "xorm span", iSpan)
	return ctx, nil
}

func (h *xormplusHook) AfterProcess(c *contexts.ContextHook) error {
	span := c.Ctx.Value("xorm span").(oteltrace.Span)
	defer span.End()
	if c.Err != nil {
		span.RecordError(c.Err)
		span.SetStatus(codes.Error, c.Err.Error())
	}
	return nil
}
