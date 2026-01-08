import * as vscode from 'vscode';
import * as cp from 'child_process';
import * as path from 'path';

// Output channel for logging
let outputChannel: vscode.OutputChannel;

export function activate(context: vscode.ExtensionContext) {
    outputChannel = vscode.window.createOutputChannel("Container Maker");
    outputChannel.appendLine('Container Maker extension is now active!');

    // Register commands
    context.subscriptions.push(
        vscode.commands.registerCommand('container-maker.checkStatus', checkStatus),
        vscode.commands.registerCommand('container-maker.up', workspaceUp),
        vscode.commands.registerCommand('container-maker.down', workspaceDown)
    );

    // Register Tree Data Providers (Scaffolding)
    const envProvider = new EnvironmentProvider();
    vscode.window.registerTreeDataProvider('cmEnvironments', envProvider);

    const svcProvider = new ServiceProvider();
    vscode.window.registerTreeDataProvider('cmServices', svcProvider);
}

export function deactivate() { }

// --- Commands ---

async function checkStatus() {
    const result = await runCMCommand('env status');
    if (result) {
        outputChannel.show(true);
        outputChannel.appendLine(result);
    }
}

async function workspaceUp() {
    vscode.window.withProgress({
        location: vscode.ProgressLocation.Notification,
        title: "Starting Workspace...",
        cancellable: false
    }, async (progress) => {
        try {
            const result = await runCMCommand('up -d');
            vscode.window.showInformationMessage('Workspace started successfully');
            outputChannel.appendLine(result || '');
        } catch (error) {
            vscode.window.showErrorMessage(`Failed to start workspace: ${error}`);
        }
    });
}

async function workspaceDown() {
    const answer = await vscode.window.showWarningMessage(
        "Are you sure you want to stop the workspace?",
        "Yes", "No"
    );

    if (answer === "Yes") {
        await runCMCommand('down');
        vscode.window.showInformationMessage('Workspace stopped');
    }
}

// --- Helper Functions ---

function getCMPath(): string {
    const config = vscode.workspace.getConfiguration('containerMaker');
    return config.get('executablePath') || 'cm';
}

function runCMCommand(args: string): Promise<string> {
    return new Promise((resolve, reject) => {
        const cmPath = getCMPath();
        const rootPath = vscode.workspace.workspaceFolders?.[0].uri.fsPath;
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

class EnvironmentProvider implements vscode.TreeDataProvider<vscode.TreeItem> {
    getTreeItem(element: vscode.TreeItem): vscode.TreeItem {
        return element;
    }

    getChildren(element?: vscode.TreeItem): Thenable<vscode.TreeItem[]> {
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

class ServiceProvider implements vscode.TreeDataProvider<vscode.TreeItem> {
    getTreeItem(element: vscode.TreeItem): vscode.TreeItem {
        return element;
    }

    getChildren(element?: vscode.TreeItem): Thenable<vscode.TreeItem[]> {
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
