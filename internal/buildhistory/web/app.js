(() => {
  "use strict";

  const elements = {
    credentialView: document.querySelector("#credential-view"),
    credentialForm: document.querySelector("#credential-form"),
    credential: document.querySelector("#credential"),
    credentialError: document.querySelector("#credential-error"),
    dashboard: document.querySelector("#dashboard"),
    disconnect: document.querySelector("#disconnect"),
    filters: document.querySelector("#filters"),
    repository: document.querySelector("#repository-filter"),
    outcome: document.querySelector("#outcome-filter"),
    search: document.querySelector("#loaded-search"),
    refresh: document.querySelector("#refresh"),
    loadMore: document.querySelector("#load-more"),
    notice: document.querySelector("#notice"),
    tableBody: document.querySelector("#build-table-body"),
    empty: document.querySelector("#empty-state"),
    loadedCount: document.querySelector("#loaded-count"),
    pageStatus: document.querySelector("#page-status"),
    historyContext: document.querySelector("#history-context"),
    matched: document.querySelector("#summary-matched"),
    successful: document.querySelector("#summary-success"),
    failed: document.querySelector("#summary-failed"),
    duration: document.querySelector("#summary-duration"),
    detailPlaceholder: document.querySelector("#detail-placeholder"),
    detailContent: document.querySelector("#detail-content")
  };

  const state = {
    token: "",
    items: [],
    nextCursor: "",
    matchedCount: 0,
    selectedID: "",
    requestVersion: 0,
    busy: false
  };

  class APIError extends Error {
    constructor(status, message) {
      super(message);
      this.status = status;
    }
  }

  elements.credentialForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    const token = elements.credential.value.trim();
    elements.credentialError.textContent = "";
    if (token.length < 32 || token.length > 512 || /\s/.test(token)) {
      elements.credentialError.textContent = "Enter the configured 32–512 character read token without spaces.";
      return;
    }
    state.token = token;
    setCredentialBusy(true);
    try {
      await loadBuilds(true);
      elements.credential.value = "";
      elements.credentialView.hidden = true;
      elements.dashboard.hidden = false;
      elements.search.focus();
    } catch (error) {
      state.token = "";
      elements.credentialError.textContent = error instanceof APIError && error.status === 401
        ? "That history read token was not accepted."
        : "Build history is unavailable. Confirm the local server and export directory.";
      elements.credential.focus();
    } finally {
      setCredentialBusy(false);
    }
  });

  elements.disconnect.addEventListener("click", () => {
    state.requestVersion += 1;
    state.token = "";
    state.items = [];
    state.nextCursor = "";
    state.matchedCount = 0;
    state.selectedID = "";
    elements.dashboard.hidden = true;
    elements.credentialView.hidden = false;
    elements.credentialError.textContent = "";
    elements.credential.value = "";
    elements.tableBody.replaceChildren();
    clearDetail();
    elements.credential.focus();
  });

  elements.filters.addEventListener("submit", async (event) => {
    event.preventDefault();
    await runBuildLoad(true);
  });

  elements.search.addEventListener("input", renderRows);
  elements.refresh.addEventListener("click", async () => {
    await runBuildLoad(true);
  });
  elements.loadMore.addEventListener("click", async () => {
    await runBuildLoad(false);
  });

  async function runBuildLoad(reset) {
    if (state.busy) {
      return;
    }
    setNotice(reset ? "Refreshing build history…" : "Loading more builds…", false);
    setDashboardBusy(true);
    try {
      await loadBuilds(reset);
      setNotice("", false);
    } catch (error) {
      if (error instanceof APIError && error.status === 401) {
        state.token = "";
        elements.dashboard.hidden = true;
        elements.credentialView.hidden = false;
        elements.credentialError.textContent = "The history read token is no longer accepted.";
        elements.credential.focus();
        return;
      }
      setNotice("Build history could not be loaded. Existing rows remain unchanged.", true);
    } finally {
      setDashboardBusy(false);
    }
  }

  async function loadBuilds(reset) {
    const version = ++state.requestVersion;
    const query = new URLSearchParams();
    query.set("limit", "100");
    const repository = elements.repository.value.trim();
    const outcome = elements.outcome.value;
    if (repository) {
      query.set("repository", repository);
    }
    if (outcome) {
      query.set("outcome", outcome);
    }
    if (!reset && state.nextCursor) {
      query.set("cursor", state.nextCursor);
    }

    const page = await fetchJSON("/api/v1/build-sessions?" + query.toString());
    if (version !== state.requestVersion) {
      return;
    }
    if (page.schemaVersion !== "buildopt.api/build-session-history/v1" ||
        !Array.isArray(page.items) || !Number.isInteger(page.matchedCount)) {
      throw new APIError(500, "Unexpected history response");
    }

    state.items = reset ? page.items : state.items.concat(page.items);
    state.nextCursor = typeof page.nextCursor === "string" ? page.nextCursor : "";
    state.matchedCount = page.matchedCount;
    if (reset) {
      state.selectedID = "";
      clearDetail();
    }
    render();
    if (reset && state.items.length > 0) {
      await selectBuild(state.items[0].id, false);
    }
  }

  async function fetchJSON(path) {
    let response;
    try {
      response = await fetch(path, {
        method: "GET",
        cache: "no-store",
        credentials: "same-origin",
        headers: {
          "Accept": "application/json",
          "Authorization": "Bearer " + state.token
        }
      });
    } catch (_error) {
      throw new APIError(0, "Local server unavailable");
    }
    if (!response.ok) {
      throw new APIError(response.status, "History request failed");
    }
    try {
      return await response.json();
    } catch (_error) {
      throw new APIError(500, "History response was not JSON");
    }
  }

  function render() {
    renderSummary();
    renderRows();
    elements.loadMore.hidden = state.nextCursor === "";
    elements.pageStatus.textContent = state.nextCursor
      ? "More matching builds are available."
      : state.items.length === 0
        ? ""
        : "All matching builds are loaded.";
    const repository = elements.repository.value.trim();
    const outcome = elements.outcome.value;
    const context = [];
    if (repository) {
      context.push("repository " + abbreviate(repository, 20));
    }
    if (outcome) {
      context.push(outcomeLabel(outcome).toLowerCase());
    }
    elements.historyContext.textContent = context.length
      ? "Newest completed builds for " + context.join(" · ") + "."
      : "Newest completed builds from private exports.";
  }

  function renderSummary() {
    const successful = state.items.filter((item) => item.outcome === "SUCCESS").length;
    const failed = state.items.filter((item) => item.outcome === "BUILD_FAILURE").length;
    const durations = state.items
      .map((item) => item.durationMs)
      .filter((value) => Number.isFinite(value) && value >= 0)
      .sort((left, right) => left - right);
    let median = null;
    if (durations.length > 0) {
      const middle = Math.floor(durations.length / 2);
      median = durations.length % 2 === 0
        ? Math.round((durations[middle - 1] + durations[middle]) / 2)
        : durations[middle];
    }
    elements.matched.textContent = formatNumber(state.matchedCount);
    elements.successful.textContent = formatNumber(successful);
    elements.failed.textContent = formatNumber(failed);
    elements.duration.textContent = median === null ? "—" : formatDuration(median);
  }

  function renderRows() {
    const term = elements.search.value.trim().toLowerCase();
    const filtered = state.items.filter((item) => {
      if (!term) {
        return true;
      }
      return String(item.id || "").toLowerCase().includes(term) ||
        String(item.revision || "").toLowerCase().includes(term);
    });
    elements.tableBody.replaceChildren(...filtered.map(buildRow));
    elements.empty.hidden = filtered.length !== 0;
    elements.loadedCount.textContent = term
      ? "Showing " + filtered.length + " of " + state.items.length + " loaded builds"
      : "Loaded " + state.items.length + " of " + state.matchedCount + " matching builds";
  }

  function buildRow(item) {
    const row = document.createElement("tr");
    if (item.id === state.selectedID) {
      row.classList.add("selected");
    }

    const outcomeCell = document.createElement("td");
    const select = document.createElement("button");
    select.type = "button";
    select.className = "build-link " + outcomeClass(item.outcome);
    select.setAttribute("aria-label", "Open " + outcomeLabel(item.outcome) + " build " + abbreviate(item.id, 18));
    const dot = document.createElement("span");
    dot.className = "status-dot";
    dot.setAttribute("aria-hidden", "true");
    select.append(dot, document.createTextNode(outcomeLabel(item.outcome)));
    select.addEventListener("click", async () => {
      await selectBuild(item.id, true);
    });
    outcomeCell.append(select);

    row.append(
      outcomeCell,
      cell(formatDate(item.completedAt)),
      cell(formatDuration(item.durationMs)),
      cell(abbreviate(item.revision, 14), "hash", item.revision),
      cell([item.environment, item.pipelineClass].filter(Boolean).join(" · ") || "—"),
      cacheCell(item.cacheState)
    );
    return row;
  }

  async function selectBuild(id, announce) {
    if (!id || !state.token) {
      return;
    }
    state.selectedID = id;
    renderRows();
    elements.detailPlaceholder.hidden = false;
    elements.detailPlaceholder.querySelector("h2").textContent = "Loading build…";
    elements.detailPlaceholder.querySelector("p").textContent = "Reading the immutable redacted session.";
    elements.detailContent.hidden = true;

    try {
      const detail = await fetchJSON("/api/v1/build-session?id=" + encodeURIComponent(id));
      if (detail.schemaVersion !== "buildopt.api/build-session-detail/v1" || !detail.session) {
        throw new APIError(500, "Unexpected detail response");
      }
      if (state.selectedID !== id) {
        return;
      }
      renderDetail(detail.session);
      if (announce) {
        setNotice("Opened build " + abbreviate(id, 18) + ".", false);
      }
    } catch (error) {
      if (state.selectedID !== id) {
        return;
      }
      clearDetail();
      elements.detailPlaceholder.querySelector("h2").textContent = "Build detail unavailable";
      elements.detailPlaceholder.querySelector("p").textContent =
        error instanceof APIError && error.status === 404
          ? "The export no longer exists."
          : "The local server could not return this build.";
      setNotice("Build detail could not be loaded.", true);
    }
  }

  function renderDetail(session) {
    const build = session.build || {};
    const workload = session.workload || {};
    const measurement = session.measurementMetadata || {};
    const assignment = session.experimentAssignment || {};
    const performance = session.performance || {};
    const invocation = Array.isArray(session.gradleInvocations)
      ? session.gradleInvocations[0] || {}
      : {};

    const header = document.createElement("header");
    header.className = "detail-header";
    const headerRow = document.createElement("div");
    headerRow.className = "detail-header-row";
    const heading = document.createElement("div");
    const eyebrow = element("p", "eyebrow", outcomeLabel(build.outcome));
    const title = element("h2", "", abbreviate(build.id, 28));
    const subtitle = element(
      "p",
      "",
      (session.complete ? "Complete" : "Recovered partial") + " · " + formatDate(build.completedAt)
    );
    heading.append(eyebrow, title, subtitle);
    const badge = element("span", "cache-pill", workload.cacheState || "Unknown cache");
    headerRow.append(heading, badge);
    header.append(headerRow);

    const sections = [
      factSection("Outcome", [
        ["Result", outcomeLabel(build.outcome)],
        ["Exit code", valueOrDash(build.exitCode)],
        ["Deliverables", valueOrDash(build.requiredDeliverablesStatus)],
        ["Plugin", valueOrDash(build.pluginVersion)]
      ]),
      factSection("Timing", [
        ["Duration", durationMeasurement(performance.customerVisibleBuildMs)],
        ["Gradle process", durationMeasurement(performance.gradleProcessUnionMs)],
        ["Started", formatDate(build.startedAt)],
        ["Completed", formatDate(build.completedAt)]
      ]),
      factSection("Workload", [
        ["Environment", valueOrDash(workload.environment)],
        ["Pipeline", valueOrDash(workload.pipelineClass)],
        ["Runner", valueOrDash(workload.runnerClass)],
        ["Workspace", valueOrDash(workload.workspaceState)],
        ["Daemon", valueOrDash(workload.daemonState)],
        ["Cache", valueOrDash(workload.cacheState)],
        ["Change", valueOrDash(workload.changeClass)]
      ]),
      factSection("Measurement", [
        ["Status", valueOrDash(measurement.status)],
        ["Clock", valueOrDash(measurement.clockSource)],
        ["Metric version", valueOrDash(measurement.metricDefinitionVersion)],
        ["Experiment arm", valueOrDash(assignment.arm)],
        ["Eligibility", valueOrDash(assignment.eligibility)]
      ])
    ];

    const requested = document.createElement("section");
    requested.className = "detail-section";
    requested.append(element("h3", "", "Requested work"));
    const invocationFacts = document.createElement("dl");
    invocationFacts.className = "fact-list";
    addFact(invocationFacts, "Gradle outcome", outcomeLabel(invocation.outcome));
    addFact(invocationFacts, "Process", durationMeasurement(invocation.processMs));
    requested.append(invocationFacts);
    const tasks = document.createElement("ul");
    tasks.className = "task-list";
    const requestedTasks = Array.isArray(invocation.requestedTasks) ? invocation.requestedTasks : [];
    for (const task of requestedTasks) {
      const item = document.createElement("li");
      item.append(element("code", "", abbreviate(task, 30)));
      tasks.append(item);
    }
    if (requestedTasks.length === 0) {
      tasks.append(element("li", "muted", "No requested tasks recorded"));
    }
    requested.append(tasks);
    sections.push(requested);

    if (session.recovery) {
      const recovery = factSection("Recovery", [
        ["Source", valueOrDash(session.recovery.source)],
        ["Recovered", formatDate(session.recovery.recoveredAt)]
      ]);
      recovery.append(element("p", "recovery-note", valueOrDash(session.recovery.reason)));
      sections.push(recovery);
    }

    elements.detailContent.replaceChildren(header, ...sections);
    elements.detailPlaceholder.hidden = true;
    elements.detailContent.hidden = false;
  }

  function factSection(title, facts) {
    const section = document.createElement("section");
    section.className = "detail-section";
    section.append(element("h3", "", title));
    const list = document.createElement("dl");
    list.className = "fact-list";
    for (const [label, value] of facts) {
      addFact(list, label, value);
    }
    section.append(list);
    return section;
  }

  function addFact(list, label, value) {
    list.append(element("dt", "", label), element("dd", "", valueOrDash(value)));
  }

  function clearDetail() {
    elements.detailContent.replaceChildren();
    elements.detailContent.hidden = true;
    elements.detailPlaceholder.hidden = false;
    elements.detailPlaceholder.querySelector("h2").textContent = "Select a build";
    elements.detailPlaceholder.querySelector("p").textContent =
      "Inspect its outcome, timing, workload, measurement, and recovery facts.";
  }

  function setCredentialBusy(busy) {
    for (const control of elements.credentialForm.elements) {
      control.disabled = busy;
    }
  }

  function setDashboardBusy(busy) {
    state.busy = busy;
    elements.refresh.disabled = busy;
    elements.loadMore.disabled = busy;
    for (const control of elements.filters.elements) {
      control.disabled = busy;
    }
  }

  function setNotice(message, error) {
    elements.notice.textContent = message;
    elements.notice.classList.toggle("error", error);
  }

  function cell(text, className, title) {
    const value = document.createElement("td");
    value.textContent = valueOrDash(text);
    if (className) {
      value.className = className;
    }
    if (title) {
      value.title = title;
    }
    return value;
  }

  function cacheCell(value) {
    const container = document.createElement("td");
    container.append(element("span", "cache-pill", valueOrDash(value)));
    return container;
  }

  function element(tag, className, text) {
    const node = document.createElement(tag);
    if (className) {
      node.className = className;
    }
    node.textContent = text;
    return node;
  }

  function valueOrDash(value) {
    return value === undefined || value === null || value === "" ? "—" : String(value);
  }

  function outcomeLabel(outcome) {
    switch (outcome) {
      case "SUCCESS":
        return "Success";
      case "BUILD_FAILURE":
        return "Build failure";
      case "CANCELLED":
        return "Cancelled";
      default:
        return valueOrDash(outcome);
    }
  }

  function outcomeClass(outcome) {
    switch (outcome) {
      case "SUCCESS":
        return "status-success";
      case "BUILD_FAILURE":
        return "status-failure";
      case "CANCELLED":
        return "status-cancelled";
      default:
        return "";
    }
  }

  function formatDate(value) {
    const date = new Date(value);
    if (!value || Number.isNaN(date.getTime())) {
      return valueOrDash(value);
    }
    return new Intl.DateTimeFormat(undefined, {
      dateStyle: "medium",
      timeStyle: "medium"
    }).format(date);
  }

  function formatDuration(value) {
    if (!Number.isFinite(value) || value < 0) {
      return "—";
    }
    if (value < 1000) {
      return Math.round(value) + " ms";
    }
    const seconds = value / 1000;
    if (seconds < 60) {
      return seconds < 10 ? seconds.toFixed(1) + " s" : Math.round(seconds) + " s";
    }
    const minutes = Math.floor(seconds / 60);
    const remainder = Math.round(seconds % 60);
    return minutes + "m " + String(remainder).padStart(2, "0") + "s";
  }

  function durationMeasurement(measurement) {
    if (!measurement || !Number.isFinite(measurement.valueMs)) {
      return measurement && measurement.state
        ? measurement.state.toLowerCase().replaceAll("_", " ")
        : "Unavailable";
    }
    return formatDuration(measurement.valueMs);
  }

  function abbreviate(value, length) {
    const text = valueOrDash(value);
    if (text.length <= length) {
      return text;
    }
    const head = Math.max(6, Math.floor((length - 1) * 0.62));
    const tail = Math.max(4, length - head - 1);
    return text.slice(0, head) + "…" + text.slice(-tail);
  }

  function formatNumber(value) {
    return Number.isFinite(value) ? new Intl.NumberFormat().format(value) : "—";
  }
})();
