# VS Code 的 WANF 语言支持

本扩展为 VS Code 中的 WANF (WJQserver's Aligned Nodal Format) 配置文件语言提供丰富的语言支持。

## ✨ 功能

*   **语法高亮**: 为关键字、注释、字符串、数字和其他语言元素提供着色，以提高可读性。
*   **代码检查 (Linting)**: 集成 `wanflint` 命令行工具，为您的 `.wanf` 文件提供实时的错误和风格问题反馈。
*   **代码格式化**: 由 `wanflint fmt` 驱动，可在保存时或手动格式化您的 `.wanf` 文件。

## 💿 安装

推荐的安装方式是通过 Visual Studio Code Marketplace。

1.  打开 **Visual Studio Code**。
2.  进入 **扩展** 视图 (`Ctrl+Shift+X`)。
3.  搜索 `WANF Language Support` 并点击 **安装**。

或者，您可以直接访问 [扩展的 Marketplace 页面](https://marketplace.visualstudio.com/items?itemName=wjqserver.wanf-language-support) 并从中安装。

### 2. 从源代码安装 (手动)

如果您想手动安装扩展或为其开发做出贡献，可以从源代码构建。

#### 先决条件

*   **Node.js**: 您必须安装 Node.js (版本 18.x 或更高) 和 `npm`。
*   **vsce**: 用于打包 VS Code 扩展的官方工具。请全局安装：
    ```sh
    npm install -g @vscode/vsce
    ```
*   **wanflint**: 为了使代码检查功能正常工作，您必须安装 `wanflint` 可执行文件并将其路径添加到系统的 PATH 环境变量中。
    ```sh
    go install github.com/WJQSERVER/wanf/wanflint@latest
    ```

#### 构建和安装步骤

1.  **克隆仓库**:
    ```sh
    git clone https://github.com/WJQSERVER/wanf.git
    ```

2.  **进入扩展目录**:
    ```sh
    cd wanf/vscode
    ```

3.  **安装依赖**:
    ```sh
    npm install
    ```

4.  **打包扩展**: 此命令会编译 TypeScript 代码并创建一个 `.vsix` 文件 (例如 `wanf-language-support-0.1.0.vsix`)。
    ```sh
    vsce package
    ```

5.  **安装 VSIX 文件**:
    *   在 VS Code 中，打开 **命令面板** (`Ctrl+Shift+P`)。
    *   运行 **扩展: 从 VSIX 安装...** 命令。
    *   选择您刚刚创建的 `.vsix` 文件。
    *   根据提示重新加载 VS Code。

## 💻 使用

安装后，当您打开带有 `.wanf` 扩展名的文件时，该扩展将自动激活。

*   **语法高亮**: 自动应用。
*   **代码检查**: 诊断和警告会自动显示在编辑器中。错误代码下方将出现下划线，您可以通过将鼠标悬停在代码上或打开“问题”面板 (`Ctrl+Shift+M`) 来查看完整消息。
*   **代码格式化**: 要格式化文件，请打开命令面板 (`Ctrl+Shift+P`) 并运行 `格式化文档`，或在编辑器设置中配置保存时格式化。

## ⚙️ 扩展设置

本扩展提供以下设置：

*   `wanf.language`: 设置 Linter 错误消息的显示语言。可选值：`auto`、`en`、`zh-cn`。默认为 `auto`。
*   `wanf.format.noSort`: 如果设置为 `true`，格式化程序将不会对块内的字段进行排序，而是保留其原始顺序。默认为 `false`。
