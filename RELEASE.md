# 发布说明流程

1. **确认版本号**：依据语义化版本设置 `VERSION`，例如 `v0.2.0`。
2. **运行测试**：
   ```bash
   go test ./...
   ```
3. **构建产物**：
   ```bash
   VERSION=v0.2.0 ./scripts/build.sh v0.2.0
   ```
   生成的 tar.gz 文件位于 `dist/`，每个压缩包中包含可执行文件与对应 SHA256 校验文件。
4. **编写发布说明**：在 GitHub Releases 或 CHANGELOG 中描述新功能、修复与已知问题，可引用 `README.md` 中的安装示例。
5. **上传产物**：将 `dist/*.tar.gz` 上传到发布页面，附带校验和。
6. **更新 todo.md**：确认任务 13 完成，准备下一个迭代。
