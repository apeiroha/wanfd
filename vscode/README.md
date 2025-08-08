# WANF Language Support for VS Code

This extension provides basic language support for the WANF (WJQserver's Aligned Nodal Format) configuration language in Visual Studio Code.

## Features

*   **Syntax Highlighting:** Provides colorization for keywords, comments, strings, numbers, and other language elements to improve readability.
*   **Linting:** Integrates with the `wanflint` command-line tool to provide real-time feedback on errors and style issues in your `.wanf` files.

## Prerequisites

For the linting feature to work, you must have the `wanflint` executable installed and available in your system's PATH. This extension calls the `wanflint lint` command to analyze your files.

## Usage

Once installed, the extension will automatically be activated when you open a file with the `.wanf` extension.

- **Syntax Highlighting:** Applied automatically.
- **Linting:** Diagnostics and warnings are automatically displayed in the editor when you open a `.wanf` file or when you save it. Errors will be underlined with a squiggle, and you can see the full error message by hovering over the underlined code or by opening the "Problems" panel in VS Code.

---
*Note: This is a foundational extension providing core language features.*
