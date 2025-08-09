# WANF Language Support for VS Code

This extension provides rich language support for the WANF (WJQserver's Aligned Nodal Format) configuration language in Visual Studio Code.

## Features

*   **Syntax Highlighting**: Provides colorization for keywords, comments, strings, numbers, and other language elements to improve readability.
*   **Linting**: Integrates with the `wanflint` command-line tool to provide real-time feedback on errors and style issues in your `.wanf` files.
*   **Formatting**: Format your `.wanf` files on save or manually, powered by `wanflint fmt`.

## Installation

The recommended way to install is through the Visual Studio Code Marketplace.

1.  Open **Visual Studio Code**.
2.  Go to the **Extensions** view (`Ctrl+Shift+X`).
3.  Search for `WANF Language Support` and click **Install**.

Alternatively, you can go directly to the [extension's Marketplace page](https://marketplace.visualstudio.com/items?itemName=wjqserver.wanf-language-support) and install from there.

### 2. Install from Source (Manual)

If you want to install the extension manually or contribute to its development, you can build it from the source.

#### Prerequisites

*   **Node.js**: You must have Node.js (version 18.x or higher) and `npm` installed.
*   **vsce**: The official tool for packaging VS Code extensions. Install it globally:
    ```sh
    npm install -g @vscode/vsce
    ```
*   **wanflint**: For the linting feature to work, you must have the `wanflint` executable installed and available in your system's PATH.
    ```sh
    go install github.com/WJQSERVER/wanf/wanflint@latest
    ```

#### Build and Install Steps

1.  **Clone the repository**:
    ```sh
    git clone https://github.com/WJQSERVER/wanf.git
    ```

2.  **Navigate to the extension directory**:
    ```sh
    cd wanf/vscode
    ```

3.  **Install dependencies**:
    ```sh
    npm install
    ```

4.  **Package the extension**: This command compiles the TypeScript code and creates a `.vsix` file (e.g., `wanf-language-support-0.1.0.vsix`).
    ```sh
    vsce package
    ```

5.  **Install the VSIX file**:
    *   In VS Code, open the **Command Palette** (`Ctrl+Shift+P`).
    *   Run the **Extensions: Install from VSIX...** command.
    *   Select the `.vsix` file you just created.
    *   Reload VS Code when prompted.

## Usage

Once installed, the extension will automatically activate when you open a file with the `.wanf` extension.

*   **Syntax Highlighting**: Applied automatically.
*   **Linting**: Diagnostics and warnings are automatically displayed in the editor. Errors will be underlined, and you can see the full message by hovering over the code or by opening the "Problems" panel (`Ctrl+Shift+M`).
*   **Formatting**: To format a file, open the Command Palette (`Ctrl+Shift+P`) and run `Format Document`, or configure format on save in your editor settings.

## Extension Settings

This extension contributes the following settings:

*   `wanf.language`: Sets the display language for linter error messages. Options: `auto`, `en`, `zh-cn`. Default is `auto`.
*   `wanf.format.noSort`: If set to `true`, the formatter will not sort fields within blocks, preserving their original order. Default is `false`.
