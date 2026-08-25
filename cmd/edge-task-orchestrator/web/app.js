const views = {
  nodes: {
    endpoint: "/api/v1/nodes",
    columns: [
      ["name", "Node"], ["region", "Region"], ["zone", "Zone"],
      ["status", "Status"], ["max_concurrent", "Slots"], ["updated_at", "Updated"]
    ]
  },
  tasks: {
    endpoint: "/api/v1/tasks",
    columns: [
      ["id", "Task"], ["status", "Status"], ["node_id", "Node"],
      ["attempt", "Attempt"], ["priority", "Priority"], ["updated_at", "Updated"]
    ]
  },
  policies: {
    endpoint: "/api/v1/policies",
    columns: [
      ["name", "Policy"], ["priority", "Priority"], ["enabled", "Enabled"],
      ["description", "Description"], ["updated_at", "Updated"]
    ]
  },
  failures: {
    endpoint: "/api/v1/dead-letters",
    columns: [
      ["id", "Dead letter"], ["execution_id", "Task"], ["node_id", "Node"],
      ["attempts", "Attempts"], ["reason", "Reason"], ["resolved", "Resolved"]
    ]
  }
};

const state = { active: "nodes", data: {}, timer: null };
const tableHead = document.querySelector("#tableHead");
const tableBody = document.querySelector("#tableBody");
const emptyState = document.querySelector("#emptyState");
const filterInput = document.querySelector("#filterInput");

function records(value) {
  if (Array.isArray(value)) return value;
  if (!value || typeof value !== "object") return [];
  for (const key of ["items", "nodes", "executions", "policies", "dead_letters", "data"]) {
    if (Array.isArray(value[key])) return value[key];
  }
  return [];
}

async function getJSON(url) {
  const response = await fetch(url, { headers: { Accept: "application/json" } });
  if (!response.ok) throw new Error(`${response.status} ${response.statusText}`);
  return response.json();
}

function displayValue(key, value) {
  if (value === null || value === undefined || value === "") return "-";
  if (typeof value === "boolean") return value ? "Yes" : "No";
  if (key.endsWith("_at")) {
    const date = new Date(value);
    if (!Number.isNaN(date.valueOf())) return date.toLocaleString();
  }
  if (typeof value === "object") return JSON.stringify(value);
  return String(value);
}

function badgeClass(value) {
  const normalized = String(value).toLowerCase();
  if (["failed", "offline", "false"].includes(normalized)) return "badge error";
  if (["pending", "queued", "running", "retrying"].includes(normalized)) return "badge warn";
  return "badge";
}

function render() {
  const view = views[state.active];
  const query = filterInput.value.trim().toLowerCase();
  const rows = (state.data[state.active] || []).filter((row) => JSON.stringify(row).toLowerCase().includes(query));
  tableHead.innerHTML = `<tr>${view.columns.map(([, label]) => `<th>${label}</th>`).join("")}</tr>`;
  tableBody.replaceChildren();
  rows.forEach((row) => {
    const tr = document.createElement("tr");
    view.columns.forEach(([key]) => {
      const td = document.createElement("td");
      const value = displayValue(key, row[key]);
      if (["status", "enabled", "resolved"].includes(key)) {
        const badge = document.createElement("span");
        badge.className = badgeClass(value);
        badge.textContent = value;
        td.appendChild(badge);
      } else {
        td.textContent = value;
      }
      tr.appendChild(td);
    });
    tableBody.appendChild(tr);
  });
  emptyState.hidden = rows.length > 0;
}

async function refresh() {
  const healthDot = document.querySelector("#healthDot");
  const healthValue = document.querySelector("#healthValue");
  try {
    const [health, nodes, tasks, policies, failures] = await Promise.all([
      getJSON("/healthz"), getJSON(views.nodes.endpoint), getJSON(views.tasks.endpoint),
      getJSON(views.policies.endpoint), getJSON(views.failures.endpoint)
    ]);
    state.data.nodes = records(nodes);
    state.data.tasks = records(tasks);
    state.data.policies = records(policies);
    state.data.failures = records(failures);
    healthDot.className = "status-dot ok";
    healthValue.textContent = health.status || "Ready";
    document.querySelector("#nodeCount").textContent = state.data.nodes.length;
    document.querySelector("#taskCount").textContent = state.data.tasks.length;
    document.querySelector("#policyCount").textContent = state.data.policies.filter((item) => item.enabled !== false).length;
    document.querySelector("#deadLetterCount").textContent = state.data.failures.filter((item) => !item.resolved).length;
    document.querySelector("#lastUpdated").textContent = `Updated ${new Date().toLocaleTimeString()}`;
    render();
  } catch (error) {
    healthDot.className = "status-dot error";
    healthValue.textContent = "Unavailable";
    document.querySelector("#lastUpdated").textContent = error.message;
  }
}

document.querySelectorAll(".tab").forEach((button) => {
  button.addEventListener("click", () => {
    document.querySelector(".tab.active").classList.remove("active");
    button.classList.add("active");
    state.active = button.dataset.view;
    filterInput.value = "";
    render();
  });
});
document.querySelector("#refreshButton").addEventListener("click", refresh);
document.querySelector("#autoRefresh").addEventListener("change", (event) => {
  clearInterval(state.timer);
  state.timer = event.target.checked ? setInterval(refresh, 10000) : null;
});
filterInput.addEventListener("input", render);
state.timer = setInterval(refresh, 10000);
refresh();
