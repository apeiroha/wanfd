import * as vscode from 'vscode';
import * as child_process from 'child_process';
import * as util from 'util';

// diagnosticCollection 用于向 VS Code 编辑器报告 linter 的错误和警告
let diagnosticCollection: vscode.DiagnosticCollection;
// extensionContext 用于访问插件的上下文, 例如获取插件的根目录路径
let extensionContext: vscode.ExtensionContext;

const exec = util.promisify(child_process.exec);

/**
 * 插件的入口点, 当插件被激活时 VS Code 会调用这个函数。
 * @param context 插件的执行上下文, 由 VS Code 提供。
 */
export function activate(context: vscode.ExtensionContext) {
	extensionContext = context;
	// 为 wanf 文件创建一个新的诊断集合, 用于在 "问题" 面板中显示错误
	diagnosticCollection = vscode.languages.createDiagnosticCollection('wanf');
	context.subscriptions.push(diagnosticCollection);

	// 当插件激活时, 如果已经有打开的文档, 则立即 lint 它
	if (vscode.window.activeTextEditor) {
		updateDiagnostics(vscode.window.activeTextEditor.document);
	}

	// 注册一系列事件监听器, 在特定事件发生时触发 lint 或格式化
	// 监听文档打开事件
	context.subscriptions.push(
		vscode.workspace.onDidOpenTextDocument(updateDiagnostics)
	);

	// 监听文档保存事件
	context.subscriptions.push(
		vscode.workspace.onDidSaveTextDocument(updateDiagnostics)
	);

	// 监听文档内容变化事件 (带防抖的 lint)
	context.subscriptions.push(
		vscode.workspace.onDidChangeTextDocument(e => updateDiagnostics(e.document))
	);

	// 为 'wanf' 语言注册一个文档格式化程序
	context.subscriptions.push(
		vscode.languages.registerDocumentFormattingEditProvider('wanf', new WanfFormattingProvider())
	);
}

/**
 * 插件被停用时调用的函数, 用于清理资源。
 */
export function deactivate() {
	diagnosticCollection.clear();
	diagnosticCollection.dispose();
}


/**
 * 实现了文档格式化功能的类。
 */
class WanfFormattingProvider implements vscode.DocumentFormattingEditProvider {
	/**
	 * 提供文档格式化编辑。通过执行 `wanflint fmt` 命令来格式化文件。
	 * @param document 需要被格式化的 VS Code 文档对象。
	 * @returns 一个 Promise, 解析为一个空的 TextEdit 数组, 因为文件是原地修改的。
	 */
	public async provideDocumentFormattingEdits(document: vscode.TextDocument): Promise<vscode.TextEdit[]> {
		try {
			const config = vscode.workspace.getConfiguration('wanf');
			const noSort = config.get<boolean>('format.noSort', false);

			let command = "wanflint fmt";
			if (noSort) {
				command += " --nosort";
			}
			command += ` ${document.fileName}`;

			console.log(`[wanf-format] Executing command: ${command}`);
			await exec(command);
			// 由于 `wanflint fmt` 是原地修改文件, VS Code 会自动检测到变化并重新加载。
			// 因此, 我们不需要返回任何编辑操作。
			return [];
		} catch (e: any) {
			console.error(`[wanf-format] Error formatting file: ${document.fileName}`, e);
			// 调用统一的错误处理函数来向用户显示错误通知
			handleExecutionError(e, 'fmt');
			return [];
		}
	}
}

/**
 * 更新指定文档的诊断信息 (即 lint 结果)。
 * @param document 需要被 lint 的 VS Code 文档对象。
 */
async function updateDiagnostics(document: vscode.TextDocument): Promise<void> {
	// 确保只处理 wanf 文件
	if (document.languageId !== 'wanf') {
		return;
	}
	console.log(`[wanf-lint] Running lint for ${document.fileName}`);

	try {
		// 使用 --json 标志运行 linter, 以便解析其结构化输出
		const command = `wanflint lint --json ${document.fileName}`;
		console.log(`[wanf-lint] Executing command: ${command}`);
		const { stdout, stderr } = await exec(command);

		if (stderr) {
			// 将 linter 的标准错误输出到控制台, 便于调试
			console.error(`[wanf-lint] linter stderr: ${stderr}`);
		}

		// 解析来自 stdout 的 JSON 输出
		const issues = JSON.parse(stdout);
		const config = vscode.workspace.getConfiguration('wanf');
		const langSetting = config.get<string>('language', 'auto');

		let useChineseTranslation = false;
		if (langSetting === 'zh-cn' || (langSetting === 'auto' && vscode.env.language.startsWith('zh'))) {
			useChineseTranslation = true;
		}

		const diagnostics: vscode.Diagnostic[] = [];
		if (issues) {
			for (const issue of issues) {
				// 将 linter 返回的 1-based 行/列号转换为 VS Code 使用的 0-based
				const range = new vscode.Range(
					issue.line - 1,
					issue.column - 1,
					issue.endLine - 1,
					issue.endColumn - 1
				);

				// 严重性由 level 决定 (0: LINT -> Error, 1: FMT -> Warning)
				const severity = issue.level === 1
					? vscode.DiagnosticSeverity.Warning
					: vscode.DiagnosticSeverity.Error;

				let finalMessage = issue.message;
				if (useChineseTranslation) {
					// 特殊处理无参数的未闭合注释错误
					if (issue.message === "unclosed block comment") {
						finalMessage = "未闭合的块注释。";
					} else {
						const template = translationMap[issue.type];
						if (template) {
							finalMessage = formatTranslation(template, issue.args);
						}
					}
				}
				diagnostics.push(new vscode.Diagnostic(range, finalMessage, severity));
			}
		}

		console.log(`[wanf-lint] Parsed diagnostics:`, diagnostics);
		// 将生成的诊断信息应用到文档中
		diagnosticCollection.set(document.uri, diagnostics);

	} catch (e: any) {
		console.error(`[wanf-lint] Error linting file: ${document.fileName}`, e);
		// 在出错时, 清除此文件的旧诊断信息, 以免残留过时的错误标记
		diagnosticCollection.set(document.uri, []);
		// 调用统一的错误处理函数来向用户显示错误通知
		handleExecutionError(e, 'lint');
	}
}

/**
 * 统一处理执行 `wanflint` 子命令时发生的异常。
 * @param error 捕获到的异常对象。
 * @param commandName 执行的子命令名称 ('lint' 或 'fmt')。
 */
function handleExecutionError(error: any, commandName: 'lint' | 'fmt'): void {
	// 检查是否是因为 'command not found' 导致的错误。
	// 这会覆盖 Linux/macOS 的 'not found' 和 Windows 的 '不是内部或外部命令' 等情况。
	if (error.code === 'ENOENT' || /not found|不是内部或外部命令/i.test(error.message)) {
		const openGuide = 'Open Install Guide';
		vscode.window.showErrorMessage(
			'wanflint command not found. Please ensure it is installed and in your PATH.',
			openGuide
		).then(selection => {
			if (selection === openGuide) {
				// 打开位于插件根目录的 README.md 文件以显示安装指南
				const readmePath = vscode.Uri.joinPath(extensionContext.extensionUri, 'README.md');
				vscode.commands.executeCommand('markdown.showPreview', readmePath);
			}
		});
	} else {
		// 对于其他执行错误 (例如, wanflint 崩溃或因致命错误返回非零退出码), 显示通用错误消息。
		// e.message 通常包含来自 stderr 的有用信息。
		vscode.window.showErrorMessage(`Error running 'wanflint ${commandName}': ${error.message}`);
	}
}

// --- 翻译模块 ---

// 此映射应与 Go linter 中的 ErrorType 枚举保持一致。
// const (
// 	ErrUnknown ErrorType = iota (0)
// 	ErrUnexpectedToken (1)
// 	ErrRedundantComma (2)
// 	ErrRedundantLabel (3)
// 	ErrUnusedVariable (4)
// 	ErrExpectDiffToken (5)
// 	ErrMissingComma (6)
// )
const translationMap: { [key: number]: string } = {
	1: "意外的标记: %s (%s)",
	2: "冗余的逗号; 块中的语句应由换行符分隔。",
	3: "块“%s”只定义了一次, 标签“%s”是多余的。",
	4: "变量“%s”已声明但从未使用。",
	5: "期望下一个标记是 %s, 但得到的是 %s",
	6: "在标记 '%s' 前缺少 ','",
};

/**
 * 使用参数格式化翻译模板字符串。
 * @param template 包含 %s 占位符的模板字符串。
 * @param args 用于替换占位符的字符串数组。
 * @returns 格式化后的字符串。
 */
function formatTranslation(template: string, args: string[] | undefined): string {
	if (!args || args.length === 0) {
		return template;
	}
	let result = template;
	for (const arg of args) {
		// 依次替换模板中的 %s
		result = result.replace("%s", arg);
	}
	return result;
}