import * as vscode from 'vscode';
import * as child_process from 'child_process';
import * as util from 'util';

// diagnosticCollection 用于报告 linter 的错误和警告
let diagnosticCollection: vscode.DiagnosticCollection;

// 插件的入口点, 当插件被激活时 VS Code 会调用这个函数
export function activate(context: vscode.ExtensionContext) {
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

// 定义了英文错误信息到中文模板的映射规则
const translationRules = [
	{
		regex: /redundant comma; statements in a block should be separated by newlines/,
		template: "冗余的逗号; 块中的语句应由换行符分隔."
	},
	{
		regex: /block "(.+?)" is defined only once, the label "(.+?)" is redundant/,
		template: "块 “%s” 只定义了一次, 标签 “%s” 是多余的."
	},
	{
		regex: /variable "(.+?)" is declared but not used/,
		template: "变量 “%s” 已声明但从未使用."
	},
	{
		regex: /unexpected token (.+?) \((.+?)\)/,
		template: "意外的标记 %s (%s)."
	}
];

// 翻译单条 linter 信息
function translateMessage(englishMessage: string): string {
	for (const rule of translationRules) {
		const matches = rule.regex.exec(englishMessage);
		if (matches) {
			// 使用 Array.prototype.slice.call(matches, 1) 来获取所有捕获组
			let translated = rule.template;
			const args = Array.prototype.slice.call(matches, 1);
			for (const arg of args) {
				translated = translated.replace('%s', arg);
			}
			return translated;
		}
	}
	// 如果没有匹配的规则, 返回原始英文信息
	return englishMessage;
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
		if (issues) {
			for (const issue of issues) {
				const lineNum = issue.line - 1; // VS Code 的行号是0索引的
				const charNum = issue.column - 1; // VS Code 的列号也是0索引的
				const endLineNum = issue.endLine - 1;
				const endCharNum = issue.endColumn - 1;
				const originalMessage = issue.message;
				const translatedMessage = translateMessage(originalMessage); // 在这里进行翻译

				const range = new vscode.Range(lineNum, charNum, endLineNum, endCharNum);
				const diagnostic = new vscode.Diagnostic(range, translatedMessage, vscode.DiagnosticSeverity.Warning);
				diagnostics.push(diagnostic);
			}
		}

		console.log(`[wanf-lint] Parsed diagnostics:`, diagnostics);
		diagnosticCollection.set(document.uri, diagnostics);

	} catch (e: any) {
		console.error(`[wanf-lint] Error linting file: ${document.fileName}`, e);
		// 检查是否是因为 'command not found' 导致的错误
		if (e.code === 'ENOENT' || /not found/i.test(e.message)) {
			vscode.window.showErrorMessage('wanflint command not found. Please ensure it is installed and in your PATH.', 'error');
			return;
		}

		// 其他错误 (例如, JSON 解析失败或 wanflint 崩溃)
		// 清除此文件的旧诊断信息
		diagnosticCollection.set(document.uri, []);
		vscode.window.showErrorMessage(`Error running wanflint: ${e.message}`);
	}
}
