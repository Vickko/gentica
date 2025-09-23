#!/bin/bash

echo "运行页面生成测试（使用真实资源）"
echo "================================"
echo ""
echo "测试说明："
echo "- 使用已有的大纲、模板和研究资料"
echo "- 生成第6页：AI的价值与机遇：个性化学习路径"
echo "- 验证agent是否充分利用研究资料"
echo ""

# 运行测试
go test -v -run TestPageGenerateWithRealResources ./ppt/agents

echo ""
echo "测试完成！"
echo ""
echo "如需查看生成的HTML文件，请查看测试日志中的工作目录。"