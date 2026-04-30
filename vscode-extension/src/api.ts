import * as http from "http";
import * as https from "https";

export interface Task {
  id: number;
  type: string;
  title: string;
  description?: string;
  status: string;
  priority: number;
  category?: string;
  phase?: string;
  created_at: string;
  updated_at: string;
  completed_at?: string;
  created_by: number;
}

export interface Project {
  id: number;
  slug: string;
  name: string;
  path: string;
  cached_todo_count: number;
  cached_doing_count: number;
  cached_total_count: number;
}

export interface ProjectStats {
  todo: number;
  doing: number;
  done: number;
  bugs_open: number;
}

export class KvikApi {
  constructor(private baseUrl: string) {}

  async getProjects(): Promise<Project[]> {
    return this.get<Project[]>("/api/v1/projects");
  }

  async getTasks(
    slug: string,
    status?: string
  ): Promise<Task[]> {
    let url = `/api/v1/projects/${slug}/tasks?limit=100`;
    if (status) {
      url += `&status=${status}`;
    }
    const data = await this.get<{ tasks: Task[] }>(url);
    return data.tasks || [];
  }

  async getStats(slug: string): Promise<ProjectStats> {
    return this.get<ProjectStats>(`/api/v1/projects/${slug}/stats`);
  }

  private get<T>(path: string): Promise<T> {
    return new Promise((resolve, reject) => {
      const url = new URL(path, this.baseUrl);
      const client = url.protocol === "https:" ? https : http;

      const options: http.RequestOptions = {
        hostname: url.hostname,
        port: url.port,
        path: url.pathname + url.search,
        method: "GET",
        headers: {},
      };

      // Support Basic Auth from URL (e.g. http://user:pass@localhost:7842)
      if (url.username && url.password) {
        const credentials = Buffer.from(
          `${decodeURIComponent(url.username)}:${decodeURIComponent(url.password)}`
        ).toString("base64");
        (options.headers as Record<string, string>)["Authorization"] = `Basic ${credentials}`;
      }

      client
        .get(options, (res) => {
          let body = "";
          res.on("data", (chunk) => (body += chunk));
          res.on("end", () => {
            if (res.statusCode && res.statusCode >= 400) {
              reject(new Error(`HTTP ${res.statusCode}: ${body}`));
              return;
            }
            try {
              resolve(JSON.parse(body));
            } catch {
              reject(new Error(`Invalid JSON: ${body}`));
            }
          });
        })
        .on("error", reject);
    });
  }
}
