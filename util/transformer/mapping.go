package transformer

import "context"

// FieldTransformer 定义字段转换接口
type FieldTransformer interface {
	Transform(ctx context.Context, doc map[string]any) (map[string]any, error)
}

// CompositeTransformer 组合多个转换器
type CompositeTransformer struct {
	transformers []FieldTransformer
}

// NewCompositeTransformer 新建一个组件转换器
func NewCompositeTransformer(transformers ...FieldTransformer) *CompositeTransformer {
	return &CompositeTransformer{transformers: transformers}
}

// Transform 组合转换器实现转换
func (ct *CompositeTransformer) Transform(ctx context.Context, doc map[string]any) (map[string]any, error) {
	var err error
	for _, t := range ct.transformers {
		doc, err = t.Transform(ctx, doc)
		if err != nil {
			return nil, err
		}
	}
	return doc, nil
}

// SimpleFieldMapper 简单字段映射器的实现
type SimpleFieldMapper struct {
	Mapping      map[string]string
	KeepOriginal bool
}

// NewSimpleFieldMapper 新建一个简单字段映射器
func NewSimpleFieldMapper(mapping map[string]string, keepOriginal bool) *SimpleFieldMapper {
	return &SimpleFieldMapper{
		Mapping:      mapping,
		KeepOriginal: keepOriginal,
	}
}

// Transform 简单的字段映射实现方法
func (m *SimpleFieldMapper) Transform(ctx context.Context, doc map[string]any) (map[string]any, error) {
	result := make(map[string]any)
	for k, v := range doc {
		if newKey, exists := m.Mapping[k]; exists {
			result[newKey] = v
			if m.KeepOriginal {
				result[k] = v
			}
		} else {
			result[k] = v
		}
	}
	return result, nil
}
