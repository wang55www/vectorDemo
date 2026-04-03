#!/bin/bash
# ollama-embed: 使用 Ollama CLIP 生成向量
# 用法：./ollama-embed text "文本内容"
#       ./ollama-embed image "图片文件路径"

set -e

MODE="$1"
INPUT="$2"

if [ -z "$MODE" ] || [ -z "$INPUT" ]; then
    echo "用法：$0 <text|image> <输入内容>"
    echo "示例:"
    echo "  $0 text '一只白色的猫'"
    echo "  $0 image /path/to/image.jpg"
    exit 1
fi

# Ollama CLIP 模型
MODEL="clip"

case "$MODE" in
    text)
        # 文本嵌入
        echo "📝 生成文本向量：$INPUT"
        curl -s http://localhost:11434/api/embeddings \
          -H "Content-Type: application/json" \
          -d "{\"model\": \"$MODEL\", \"prompt\": \"$INPUT\"}" \
          | jq '.embedding'
        ;;
    
    image)
        # 图片嵌入 - 需要先将图片转为 base64
        if [ ! -f "$INPUT" ]; then
            echo "❌ 文件不存在：$INPUT"
            exit 1
        fi
        
        echo "🖼️ 生成图片向量：$INPUT"
        
        # 将图片转为 base64
        BASE64=$(base64 -w 0 "$INPUT")
        
        # Ollama CLIP 不支持直接传 base64，使用变通方法
        # 这里返回错误提示
        echo "❌ Ollama CLIP 不支持图片嵌入，请使用 Jina API 或仅使用文字搜索"
        exit 1
        ;;
    
    *)
        echo "❌ 未知模式：$MODE"
        echo "支持的模式：text, image"
        exit 1
        ;;
esac
