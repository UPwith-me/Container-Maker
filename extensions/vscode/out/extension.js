"use strict";
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.activate = activate;
exports.deactivate = deactivate;
const vscode = require("vscode");
const cp = require("child_process");
// Output channel for logging
let outputChannel;
function activate(context) {
    outputChannel = vscode.window.createOutputChannel("Container Maker");
    outputChannel.appendLine('Container Maker extension is now active!');
    // Register commands
    context.subscriptions.push(vscode.commands.registerCommand('container-maker.checkStatus', checkStatus), vscode.commands.registerCommand('container-maker.up', workspaceUp), vscode.commands.registerCommand('container-maker.down', workspaceDown));
    // Register Tree Data Providers (Scaffolding)
    const envProvider = new EnvironmentProvider();
    vscode.window.registerTreeDataProvider('cmEnvironments', envProvider);
    const svcProvider = new ServiceProvider();
    vscode.window.registerTreeDataProvider('cmServices', svcProvider);
}
function deactivate() { }
// --- Commands ---
function checkStatus() {
    return __awaiter(this, void 0, void 0, function* () {
        const result = yield runCMCommand('env status');
        if (result) {
            outputChannel.show(true);
            outputChannel.appendLine(result);
        }
    });
}
function workspaceUp() {
    return __awaiter(this, void 0, void 0, function* () {
        vscode.window.withProgress({
            location: vscode.ProgressLocation.Notification,
            title: "Starting Workspace...",
            cancellable: false
        }, (progress) => __awaiter(this, void 0, void 0, function* () {
            try {
                const result = yield runCMCommand('up -d');
                vscode.window.showInformationMessage('Workspace started successfully');
                outputChannel.appendLine(result || '');
            }
            catch (error) {
                vscode.window.showErrorMessage(`Failed to start workspace: ${error}`);
            }
        }));
    });
}
function workspaceDown() {
    return __awaiter(this, void 0, void 0, function* () {
        const answer = yield vscode.window.showWarningMessage("Are you sure you want to stop the workspace?", "Yes", "No");
        if (answer === "Yes") {
            yield runCMCommand('down');
            vscode.window.showInformationMessage('Workspace stopped');
        }
    });
}
// --- Helper Functions ---
function getCMPath() {
    const config = vscode.workspace.getConfiguration('containerMaker');
    return config.get('executablePath') || 'cm';
}
function runCMCommand(args) {
    return new Promise((resolve, reject) => {
        var _a;
        const cmPath = getCMPath();
        const rootPath = (_a = vscode.workspace.workspaceFolders) === null || _a === void 0 ? void 0 : _a[0].uri.fsPath;
        const cwd = rootPath || process.cwd();
        const cmd = `"${cmPath}" ${args}`;
        outputChannel.appendLine(`> Executing: ${cmd} in ${cwd}`);
        cp.exec(cmd, { cwd }, (err, stdout, stderr) => {
            if (err) {
                outputChannel.appendLine(`Error: ${err.message}`);
                outputChannel.appendLine(`Stderr: ${stderr}`);
                reject(stderr || err.message);
                return;
            }
            resolve(stdout);
        });
    });
}
// --- Tree View Scaffolding ---
class EnvironmentProvider {
    getTreeItem(element) {
        return element;
    }
    getChildren(element) {
        // Mock data for scaffolding
        if (!element) {
            return Promise.resolve([
                new vscode.TreeItem("Dev Environment", vscode.TreeItemCollapsibleState.None),
                new vscode.TreeItem("Staging Environment", vscode.TreeItemCollapsibleState.None)
            ]);
        }
        return Promise.resolve([]);
    }
}
class ServiceProvider {
    getTreeItem(element) {
        return element;
    }
    getChildren(element) {
        // Mock data for scaffolding
        if (!element) {
            return Promise.resolve([
                new vscode.TreeItem("frontend (Node.js)", vscode.TreeItemCollapsibleState.None),
                new vscode.TreeItem("backend (Go)", vscode.TreeItemCollapsibleState.None),
                new vscode.TreeItem("database (Postgres)", vscode.TreeItemCollapsibleState.None)
            ]);
        }
        return Promise.resolve([]);
    }
}
//# sourceMappingURL=extension.js.map