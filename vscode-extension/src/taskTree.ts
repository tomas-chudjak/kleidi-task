import * as vscode from "vscode";
import { KvikApi, Task, Project } from "./api";

export class TaskTreeProvider implements vscode.TreeDataProvider<TaskItem> {
  private _onDidChangeTreeData = new vscode.EventEmitter<
    TaskItem | undefined
  >();
  readonly onDidChangeTreeData = this._onDidChangeTreeData.event;

  private statusFilter: string | undefined = "todo";
  private projects: Project[] = [];

  constructor(private api: KvikApi) {}

  refresh(): void {
    this._onDidChangeTreeData.fire(undefined);
  }

  setFilter(status: string | undefined): void {
    this.statusFilter = status;
    this.refresh();
  }

  getTreeItem(element: TaskItem): vscode.TreeItem {
    return element;
  }

  async getChildren(element?: TaskItem): Promise<TaskItem[]> {
    if (!element) {
      // Root level: show projects
      try {
        this.projects = await this.api.getProjects();
      } catch {
        return [
          new TaskItem(
            "Server not running",
            "",
            vscode.TreeItemCollapsibleState.None,
            "error"
          ),
          new TaskItem(
            "Run: kvt serve",
            "",
            vscode.TreeItemCollapsibleState.None,
            "hint"
          ),
          new TaskItem(
            "Default: http://localhost:7842",
            "",
            vscode.TreeItemCollapsibleState.None,
            "hint"
          ),
        ];
      }

      if (this.projects.length === 0) {
        return [
          new TaskItem(
            "No projects found",
            "",
            vscode.TreeItemCollapsibleState.None,
            "info"
          ),
          new TaskItem(
            "Run: kvt init",
            "",
            vscode.TreeItemCollapsibleState.None,
            "hint"
          ),
        ];
      }

      if (this.projects.length === 1) {
        // Single project: show tasks directly
        return this.getTaskItems(this.projects[0].slug);
      }

      return this.projects.map(
        (p) =>
          new TaskItem(
            p.name,
            p.slug,
            vscode.TreeItemCollapsibleState.Collapsed,
            "project"
          )
      );
    }

    // Expand task → show description lines
    if (element.itemType === "task" && element.task) {
      return this.getTaskDetail(element.task);
    }

    if (element.itemType === "project") {
      return this.getTaskItems(element.slug);
    }

    return [];
  }

  private async getTaskItems(slug: string): Promise<TaskItem[]> {
    try {
      const tasks = await this.api.getTasks(slug, this.statusFilter);
      if (tasks.length === 0) {
        return [
          new TaskItem(
            "No tasks",
            slug,
            vscode.TreeItemCollapsibleState.None,
            "info"
          ),
        ];
      }
      return tasks.map((t) => TaskItem.fromTask(t, slug));
    } catch {
      return [
        new TaskItem(
          "Error loading tasks",
          slug,
          vscode.TreeItemCollapsibleState.None,
          "error"
        ),
      ];
    }
  }

  private getTaskDetail(task: Task): TaskItem[] {
    const items: TaskItem[] = [];

    // Status and type info
    items.push(
      new TaskItem(
        `Status: ${task.status}`,
        "",
        vscode.TreeItemCollapsibleState.None,
        "detail"
      )
    );
    items.push(
      new TaskItem(
        `Type: ${task.type} · Priority: ${task.priority}`,
        "",
        vscode.TreeItemCollapsibleState.None,
        "detail"
      )
    );

    if (task.category) {
      items.push(
        new TaskItem(
          `Category: ${task.category}`,
          "",
          vscode.TreeItemCollapsibleState.None,
          "detail"
        )
      );
    }

    if (task.phase) {
      items.push(
        new TaskItem(
          `Phase: ${task.phase}`,
          "",
          vscode.TreeItemCollapsibleState.None,
          "detail"
        )
      );
    }

    // Description lines
    if (task.description) {
      items.push(
        new TaskItem(
          "───",
          "",
          vscode.TreeItemCollapsibleState.None,
          "detail"
        )
      );
      const lines = task.description.split("\n").filter((l) => l.trim() !== "");
      for (const line of lines.slice(0, 10)) {
        items.push(
          new TaskItem(
            line.trim(),
            "",
            vscode.TreeItemCollapsibleState.None,
            "detail"
          )
        );
      }
      if (lines.length > 10) {
        items.push(
          new TaskItem(
            `... (${lines.length - 10} more lines)`,
            "",
            vscode.TreeItemCollapsibleState.None,
            "detail"
          )
        );
      }
    }

    return items;
  }

  getProjects(): Project[] {
    return this.projects;
  }
}

const statusIcons: Record<string, string> = {
  todo: "circle-outline",
  doing: "play-circle",
  done: "check",
};

export class TaskItem extends vscode.TreeItem {
  constructor(
    public readonly label: string,
    public readonly slug: string,
    public readonly collapsibleState: vscode.TreeItemCollapsibleState,
    public readonly itemType: "task" | "project" | "info" | "error" | "detail" | "hint",
    public readonly task?: Task
  ) {
    super(label, collapsibleState);

    if (itemType === "task" && task) {
      this.description = `#${task.id} · ${task.type} · P${task.priority}`;
      this.tooltip = `#${task.id} ${task.title}\nType: ${task.type} | Status: ${task.status} | Priority: ${task.priority}\n\nClick to expand details`;
      this.iconPath = new vscode.ThemeIcon(
        statusIcons[task.status] || "circle-outline"
      );
      this.contextValue = "task";
    } else if (itemType === "project") {
      this.iconPath = new vscode.ThemeIcon("folder");
      this.contextValue = "project";
    } else if (itemType === "detail") {
      this.iconPath = new vscode.ThemeIcon("dash");
    } else if (itemType === "error") {
      this.iconPath = new vscode.ThemeIcon("error");
    } else if (itemType === "hint") {
      this.iconPath = new vscode.ThemeIcon("terminal");
    } else {
      this.iconPath = new vscode.ThemeIcon("info");
    }
  }

  static fromTask(task: Task, slug: string): TaskItem {
    const hasDetail = !!(task.description || task.category || task.phase);
    return new TaskItem(
      task.title,
      slug,
      hasDetail
        ? vscode.TreeItemCollapsibleState.Collapsed
        : vscode.TreeItemCollapsibleState.None,
      "task",
      task
    );
  }
}
