package pkservice

// ptr 返回 v 的地址，用于构造 dev 侧指针字段的实体 fixture。
func ptr[T any](v T) *T { return &v }
