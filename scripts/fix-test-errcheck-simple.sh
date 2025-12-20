#!/bin/bash
# 简单批量修复测试文件中的 errcheck 错误

set -e

echo "批量修复测试文件中的 errcheck 错误..."

# 获取所有有 errcheck 错误的测试文件
FILES=$(golangci-lint run 2>&1 | grep "errcheck" | grep "_test.go:" | cut -d: -f1 | sort -u)

count=0
for file in $FILES; do
    echo "处理: $file"

    # 备份文件
    cp "$file" "$file.bak"

    # 修复常见模式（macOS 兼容）
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS sed
        # 1. defer xxx.Close() 不需要修复（已经是 defer）
        # 2. 行首的方法调用添加 _ =
        sed -i '' 's/^\([[:space:]]*\)\([a-zA-Z_][a-zA-Z0-9_]*\)\.\(Write\|Update\|Save\|Add\|Share\|Track\|Remove\|RemoveAll\|Setenv\|Unsetenv\|Join\|Broadcast\|Every\|Shutdown\|Terminate\|Read\|Decode\|Marshal\)(/\1_ = \2.\3(/g' "$file"

        # 3. defer 后面的 Close/Shutdown/RemoveAll
        sed -i '' 's/defer \([a-zA-Z_][a-zA-Z0-9_]*\)\.\(Close\|Shutdown\|RemoveAll\)(/defer func() { _ = \1.\2(/g' "$file"
        sed -i '' 's/defer func() { _ = \([a-zA-Z_][a-zA-Z0-9_]*\)\.\(Close\|Shutdown\|RemoveAll\)(/defer func() { _ = \1.\2(/g' "$file"
    else
        # Linux sed
        sed -i 's/^\([[:space:]]*\)\([a-zA-Z_][a-zA-Z0-9_]*\)\.\(Write\|Update\|Save\|Add\|Share\|Track\|Remove\|RemoveAll\|Setenv\|Unsetenv\|Join\|Broadcast\|Every\|Shutdown\|Terminate\|Read\|Decode\|Marshal\)(/\1_ = \2.\3(/g' "$file"
        sed -i 's/defer \([a-zA-Z_][a-zA-Z0-9_]*\)\.\(Close\|Shutdown\|RemoveAll\)(/defer func() { _ = \1.\2(/g' "$file"
        sed -i 's/defer func() { _ = \([a-zA-Z_][a-zA-Z0-9_]*\)\.\(Close\|Shutdown\|RemoveAll\)(/defer func() { _ = \1.\2(/g' "$file"
    fi

    # 检查是否有改动
    if ! diff -q "$file" "$file.bak" > /dev/null 2>&1; then
        count=$((count + 1))
        echo "  ✓ 已修复"
    else
        echo "  - 无需修复"
    fi

    # 删除备份
    rm "$file.bak"
done

echo ""
echo "修复完成！共处理 $count 个文件"
echo ""
echo "运行 golangci-lint 检查结果..."
remaining=$(golangci-lint run 2>&1 | grep "errcheck" | wc -l | tr -d ' ')
echo "剩余 errcheck 错误: $remaining"
