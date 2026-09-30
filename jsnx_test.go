package jsnx

import (
	"fmt"
	"testing"
)

func BenchmarkGetString(b *testing.B) {
	str := `{"list":[{"cls":{"name": "张三", "sex": "男", "age": 26}},{"cls":{"name": "李四", "age": 30, "sex": "女"}}]}`
	holder, err := Parse(str)
	if err != nil {
		fmt.Printf("Err: %v\n", err)
		return
	}

	//b.ResetTimer()

	for b.Loop() {
		// str := `{"list":[{"cls":{"name": "张三", "sex": "男", "age": 26}},{"cls":{"name": "李四", "age": 30, "sex": "女"}}]}`
		// holder, _ := Parse(str)
		// holder.String("", "")
		holder.GetString("list/1/name")
	}

}
