import * as vscode from "vscode";
import { KvikApi } from "./api";
import { TaskTreeProvider, TaskItem } from "./taskTree";

let statusBarItem: vscode.StatusBarItem;
let refreshTimer: NodeJS.Timeout | undefined;

export function activate(context: vscode.ExtensionContext) {
  const config = vscode.workspace.getConfiguration("kvikTasks");
  const serverUrl = config.get<string>("serverUrl", "http://localhost:7842");
  const refreshInterval = config.get<number>("refreshInterval", 10);

  const api = new KvikApi(serverUrl);
  const treeProvider = new TaskTreeProvider(api);

  // Register tree view
  const treeView = vscode.window.createTreeView("kvikTasks", {
    treeDataProvider: treeProvider,
    showCollapseAll: true,
  });
  context.subscriptions.push(treeView);

  // Status bar
  statusBarItem = vscode.window.createStatusBarItem(
    vscode.StatusBarAlignment.Left,
    50
  );
  statusBarItem.command = "kvikTasks.filterAll";
  statusBarItem.tooltip = "Kvik Tasks — click to show all";
  context.subscriptions.push(statusBarItem);

  // Commands
  context.subscriptions.push(
    vscode.commands.registerCommand("kvikTasks.refresh", () => {
      treeProvider.refresh();
      updateStatusBar(api);
    }),

    vscode.commands.registerCommand("kvikTasks.filterTodo", () => {
      treeProvider.setFilter("todo");
    }),

    vscode.commands.registerCommand("kvikTasks.filterDoing", () => {
      treeProvider.setFilter("doing");
    }),

    vscode.commands.registerCommand("kvikTasks.filterAll", () => {
      treeProvider.setFilter(undefined);
    }),

    vscode.commands.registerCommand(
      "kvikTasks.insertToTerminal",
      (item: TaskItem) => {
        if (!item.task) return;
        const text = `task: #${item.task.id} ${item.task.title} (${item.task.type}, P${item.task.priority}, ${item.task.status})`;
        const terminal =
          vscode.window.activeTerminal ||
          vscode.window.createTerminal("Kvik Tasks");
        terminal.show();
        terminal.sendText(text, false);
      }
    ),

    vscode.commands.registerCommand(
      "kvikTasks.openInBrowser",
      (item: TaskItem) => {
        if (!item.task) return;
        const url = `${serverUrl}/p/${item.slug}/t/${item.task.id}`;
        vscode.env.openExternal(vscode.Uri.parse(url));
      }
    ),

    vscode.commands.registerCommand(
      "kvikTasks.copyReference",
      (item: TaskItem) => {
        if (!item.task) return;
        vscode.env.clipboard.writeText(`#${item.task.id}`);
        vscode.window.showInformationMessage(
          `Copied #${item.task.id} to clipboard`
        );
      }
    )
  );

  // Auto-refresh
  if (refreshInterval > 0) {
    refreshTimer = setInterval(() => {
      treeProvider.refresh();
      updateStatusBar(api);
    }, refreshInterval * 1000);
  }

  // Initial load
  updateStatusBar(api);

  // Config change listener
  context.subscriptions.push(
    vscode.workspace.onDidChangeConfiguration((e) => {
      if (e.affectsConfiguration("kvikTasks")) {
        vscode.window.showInformationMessage(
          "Kvik Tasks: Reload window to apply config changes"
        );
      }
    })
  );
}

async function updateStatusBar(api: KvikApi) {
  try {
    const projects = await api.getProjects();
    let todo = 0;
    let doing = 0;
    for (const p of projects) {
      try {
        const stats = await api.getStats(p.slug);
        todo += stats.todo;
        doing += stats.doing;
      } catch {
        // skip unreachable projects
      }
    }
    statusBarItem.text = `$(checklist) kvt: ${todo} todo · ${doing} doing`;
    statusBarItem.show();
  } catch {
    statusBarItem.text = "$(checklist) kvt: offline";
    statusBarItem.show();
  }
}

export function deactivate() {
  if (refreshTimer) {
    clearInterval(refreshTimer);
  }
}
