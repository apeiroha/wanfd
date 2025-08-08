import * as vscode from 'vscode';
import * as child_process from 'child_process';
import * as util from 'util';

// diagnosticCollection 用于报告 linter 的错误和警告
let diagnosticCollection: vscode.DiagnosticCollection;
let extensionContext: vscode.ExtensionContext;

// 插件的入口点, 当插件被激活时 VS Code 会调用这个函数
export function activate(context: vscode.ExtensionContext) {
	extensionContext = context;
	// 为 wanf 文件创建一个新的诊断集合
	diagnosticCollection = vscode.languages.createDiagnosticCollection('wanf');
	context.subscriptions.push(diagnosticCollection);

	// 当插件激活时, 如果已经有打开的文档, 则立即 lint 它
	if (vscode.window.activeTextEditor) {
		updateDiagnostics(vscode.window.activeTextEditor.document);
	}

	// 监听文档打开事件, 在打开时进行 lint
	context.subscriptions.push(
		vscode.workspace.onDidOpenTextDocument(updateDiagnostics)
	);

	// 监听文档保存事件, 在保存时进行 lint
	context.subscriptions.push(
		vscode.workspace.onDidSaveTextDocument(updateDiagnostics)
	);

	// 监听文档内容变化事件, 在变化时进行 lint
	context.subscriptions.push(
		vscode.workspace.onDidChangeTextDocument(e => updateDiagnostics(e.document))
	);

	// 注册格式化程序
	context.subscriptions.push(
		vscode.languages.registerDocumentFormattingEditProvider('wanf', new WanfFormattingProvider())
	);
}

class WanfFormattingProvider implements vscode.DocumentFormattingEditProvider {
	public async provideDocumentFormattingEdits(document: vscode.TextDocument): Promise<vscode.TextEdit[]> {
		try {
			const command = `wanflint fmt ${document.fileName}`;
			console.log(`[wanf-format] Executing command: ${command}`);
			await exec(command);
			// The `wanflint fmt` command modifies the file in-place.
			// VS Code will automatically detect the file change and reload it.
			// Therefore, we don't need to return any edits.
			return [];
		} catch (e: any) {
			console.error(`[wanf-format] Error formatting file: ${document.fileName}`, e);
			vscode.window.showErrorMessage(`Error running 'wanflint fmt': ${e.message}`);
			return [];
		}
	}
}

// 当插件被停用时, VS Code 会调用这个函数
export function deactivate() { }

const exec = util.promisify(child_process.exec);

// --- 翻译模块 ---

// Go linter 中的 ErrorType 枚举
// const (
// 	ErrUnknown ErrorType = iota (0)
// 	ErrUnexpectedToken (1)
// 	ErrRedundantComma (2)
// 	ErrRedundantLabel (3)
// 	ErrUnusedVariable (4)
// )
const translationMap: { [key: number]: string } = {
	1: "意外的标记: %s (%s)",
	2: "冗余的逗号; 块中的语句应由换行符分隔。",
	3: "块“%s”只定义了一次, 标签“%s”是多余的。",
	4: "变量“%s”已声明但从未使用。",
};

function formatTranslation(template: string, args: string[] | undefined): string {
	if (!args || args.length === 0) {
		return template;
	}
	let result = template;
	for (const arg of args) {
		result = result.replace("%s", arg);
	}
	return result;
}

async function updateDiagnostics(document: vscode.TextDocument): Promise<void> {
	if (document.languageId !== 'wanf') {
		return;
	}
	console.log(`[wanf-lint] Running lint for ${document.fileName}`);

	const diagnostics: vscode.Diagnostic[] = [];
	try {
		// 使用 --json 标志运行 linter, 输出将是 JSON 格式.
		const command = `wanflint lint --json ${document.fileName}`;
		console.log(`[wanf-lint] Executing command: ${command}`);
		const { stdout, stderr } = await exec(command);

		if (stderr) {
			console.error(`[wanf-lint] linter stderr: ${stderr}`);
		}

		// 解析来自 stdout 的 JSON 输出
		const issues = JSON.parse(stdout);
		const config = vscode.workspace.getConfiguration('wanf');
		const langSetting = config.get<string>('language');

		let useChineseTranslation = false;
		if (langSetting === 'zh-cn') {
			useChineseTranslation = true;
		} else if (langSetting === 'auto' && vscode.env.language.startsWith('zh')) {
			useChineseTranslation = true;
		}


		if (issues) {
			for (const issue of issues) {
				const lineNum = issue.line - 1;
				const charNum = issue.column - 1;
				const endLineNum = issue.endLine - 1;
				const endCharNum = issue.endColumn - 1;
				const range = new vscode.Range(lineNum, charNum, endLineNum, endCharNum);

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

				const diagnostic = new vscode.Diagnostic(range, finalMessage, severity);
				diagnostics.push(diagnostic);
			}
		}

		console.log(`[wanf-lint] Parsed diagnostics:`, diagnostics);
		diagnosticCollection.set(document.uri, diagnostics);

	} catch (e: any) {
		console.error(`[wanf-lint] Error linting file: ${document.fileName}`, e);
		// 检查是否是因为 'command not found' 导致的错误
		if (e.code === 'ENOENT' || /not found/i.test(e.message)) {
			const openGuide = 'Open Install Guide';
			vscode.window.showErrorMessage(
				'wanflint command not found. Please ensure it is installed and in your PATH.',
				openGuide
			).then(selection => {
				if (selection === openGuide) {
					// README.md 在 vscode 插件目录的根目录中
					const readmePath = vscode.Uri.joinPath(extensionContext.extensionUri, 'README.md');
					vscode.commands.executeCommand('markdown.showPreview', readmePath);
				}
			});
			return;
		}

		// 其他错误 (例如, JSON 解析失败或 wanflint 崩溃)
		// 清除此文件的旧诊断信息
		diagnosticCollection.set(document.uri, []);
		vscode.window.showErrorMessage(`Error running wanflint: ${e.message}`);
	}
}
