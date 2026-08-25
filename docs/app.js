const DATA_URL = "./data/go-tool-sizes.json";
const MEBIBYTE = 1024 * 1024;

const COLORS = {
  ink: "#172126",
  inkSoft: "#526168",
  line: "#d8e1e4",
  cyan: "#00add8",
  cyanFill: "rgba(0, 173, 216, 0.12)",
  coral: "#ef6f61",
  coralFill: "rgba(239, 111, 97, 0.12)",
  yellow: "#e3b341",
  green: "#319b78",
};

const OS_NAMES = {
  aix: "AIX",
  darwin: "macOS",
  dragonfly: "DragonFly BSD",
  freebsd: "FreeBSD",
  illumos: "illumos",
  linux: "Linux",
  netbsd: "NetBSD",
  openbsd: "OpenBSD",
  plan9: "Plan 9",
  solaris: "Solaris",
  windows: "Windows",
};

const ARCH_NAMES = {
  "386": "386",
  amd64: "AMD64",
  arm: "ARM",
  arm64: "ARM64",
  armv6l: "ARMv6",
  loong64: "Loong64",
  mips: "MIPS",
  mips64: "MIPS64",
  mips64le: "MIPS64LE",
  mipsle: "MIPSLE",
  ppc64: "PPC64",
  ppc64le: "PPC64LE",
  riscv64: "RISC-V 64",
  s390x: "s390x",
};

const elements = {
  status: document.querySelector("#status"),
  metricGrid: document.querySelector("#metricGrid"),
  visuals: document.querySelector("#visuals"),
  platformSelect: document.querySelector("#platformSelect"),
  releaseSelect: document.querySelector("#releaseSelect"),
  toolSelect: document.querySelector("#toolSelect"),
  downloadCsv: document.querySelector("#downloadCsv"),
  generatedAt: document.querySelector("#generatedAt"),
  datasetRange: document.querySelector("#datasetRange"),
  trendTarget: document.querySelector("#trendTarget"),
  latestArchive: document.querySelector("#latestArchive"),
  latestRelease: document.querySelector("#latestRelease"),
  archiveChange: document.querySelector("#archiveChange"),
  firstRelease: document.querySelector("#firstRelease"),
  peakArchive: document.querySelector("#peakArchive"),
  peakRelease: document.querySelector("#peakRelease"),
  releaseCoverage: document.querySelector("#releaseCoverage"),
  archiveCount: document.querySelector("#archiveCount"),
  snapshotTitle: document.querySelector("#snapshotTitle"),
  snapshotTotal: document.querySelector("#snapshotTotal"),
  topTools: document.querySelector("#topTools"),
  toolChartTitle: document.querySelector("#toolChartTitle"),
  toolCoverage: document.querySelector("#toolCoverage"),
  toolColumnHeader: document.querySelector("#toolColumnHeader"),
  releaseTable: document.querySelector("#releaseTable"),
  tableCount: document.querySelector("#tableCount"),
};

const state = {
  report: null,
  releases: [],
  platform: "linux/amd64",
  release: "",
  tool: "compile",
  rows: [],
  charts: {},
};

document.addEventListener("DOMContentLoaded", initialize);

async function initialize() {
  if (window.lucide) {
    window.lucide.createIcons();
  }

  try {
    if (!window.Chart) {
      throw new Error("Chart library did not load");
    }
    const response = await fetch(DATA_URL);
    if (!response.ok) {
      throw new Error(`Dataset request failed with ${response.status}`);
    }
    state.report = await response.json();
    validateReport(state.report);
    state.releases = [...state.report.releases].sort(compareReleases);

    configureChartDefaults();
    populatePlatformSelect();
    bindControls();
    updatePlatform();
    updateDatasetMeta();

    elements.status.hidden = true;
    elements.metricGrid.hidden = false;
    elements.visuals.hidden = false;
    elements.platformSelect.disabled = false;
    elements.releaseSelect.disabled = false;
    elements.toolSelect.disabled = false;
    elements.downloadCsv.disabled = false;
  } catch (error) {
    showError(error);
  }
}

function validateReport(report) {
  if (report?.schema_version !== 1 || !Array.isArray(report.releases) || report.releases.length === 0) {
    throw new Error("Unsupported or empty dataset");
  }
}

function configureChartDefaults() {
  Chart.defaults.color = COLORS.inkSoft;
  Chart.defaults.font.family = "IBM Plex Sans, sans-serif";
  Chart.defaults.font.size = 11;
  Chart.defaults.animation.duration = window.matchMedia("(prefers-reduced-motion: reduce)").matches ? 0 : 450;
  Chart.defaults.plugins.legend.labels.boxWidth = 12;
  Chart.defaults.plugins.legend.labels.boxHeight = 3;
  Chart.defaults.plugins.legend.labels.usePointStyle = true;
  Chart.defaults.plugins.legend.labels.pointStyle = "line";
}

function bindControls() {
  elements.platformSelect.addEventListener("change", () => {
    state.platform = elements.platformSelect.value;
    updatePlatform();
  });
  elements.releaseSelect.addEventListener("change", () => {
    state.release = elements.releaseSelect.value;
    updateSnapshot();
    updateTable();
  });
  elements.toolSelect.addEventListener("change", () => {
    state.tool = elements.toolSelect.value;
    updateToolChart();
    updateTable();
  });
  elements.downloadCsv.addEventListener("click", downloadCsv);
}

function populatePlatformSelect() {
  const coverage = new Map();
  for (const release of state.releases) {
    const seen = new Set();
    for (const platform of release.platforms) {
      const key = platformKey(platform);
      if (!seen.has(key)) {
        coverage.set(key, (coverage.get(key) || 0) + 1);
        seen.add(key);
      }
    }
  }

  const keys = [...coverage.keys()].sort((left, right) => {
    const coverageDifference = coverage.get(right) - coverage.get(left);
    return coverageDifference || platformLabel(left).localeCompare(platformLabel(right));
  });
  elements.platformSelect.replaceChildren(...keys.map((key) => {
    const option = document.createElement("option");
    option.value = key;
    option.textContent = `${platformLabel(key)} · ${coverage.get(key)}/${state.releases.length}`;
    return option;
  }));

  if (!coverage.has(state.platform)) {
    state.platform = keys[0];
  }
  elements.platformSelect.value = state.platform;
}

function updatePlatform() {
  state.rows = state.releases.map((release) => makeRow(release, preferredPlatform(release, state.platform)));
  const availableRows = state.rows.filter((row) => row.platform);
  state.release = availableRows.at(-1)?.version || "";

  populateReleaseSelect(availableRows);
  populateToolSelect(availableRows);
  updateMetrics(availableRows);
  updateArchiveChart();
  updateCompositionChart();
  updateToolChart();
  updateSnapshot();
  updateTable();
  elements.trendTarget.textContent = platformLabel(state.platform);
}

function populateReleaseSelect(rows) {
  elements.releaseSelect.replaceChildren(...[...rows].reverse().map((row) => {
    const option = document.createElement("option");
    option.value = row.version;
    option.textContent = row.version;
    return option;
  }));
  elements.releaseSelect.value = state.release;
}

function populateToolSelect(rows) {
  const coverage = new Map();
  for (const row of rows) {
    const names = new Set(row.platform.tools.map((tool) => tool.name));
    for (const name of names) {
      coverage.set(name, (coverage.get(name) || 0) + 1);
    }
  }

  const names = [...coverage.keys()].sort((left, right) => {
    const coverageDifference = coverage.get(right) - coverage.get(left);
    return coverageDifference || left.localeCompare(right);
  });
  elements.toolSelect.replaceChildren(...names.map((name) => {
    const option = document.createElement("option");
    option.value = name;
    option.textContent = `${name} · ${coverage.get(name)}/${rows.length}`;
    return option;
  }));

  if (!coverage.has(state.tool)) {
    state.tool = coverage.has("go") ? "go" : names[0];
  }
  elements.toolSelect.value = state.tool;
}

function makeRow(release, platform) {
  if (!platform) {
    return { version: release.version, platform: null };
  }
  const commands = sum(platform.tools.filter((tool) => tool.category === "command").map((tool) => tool.size));
  const internalTools = sum(platform.tools.filter((tool) => tool.category === "tool").map((tool) => tool.size));
  return {
    version: release.version,
    platform,
    archive: platform.archive.size,
    commands,
    internalTools,
    executablePayload: commands + internalTools,
    toolCount: platform.tools.length,
  };
}

function updateMetrics(rows) {
  const first = rows[0];
  const latest = rows.at(-1);
  const peak = rows.reduce((largest, row) => row.archive > largest.archive ? row : largest, rows[0]);
  const change = ((latest.archive / first.archive) - 1) * 100;

  elements.latestArchive.textContent = formatBytes(latest.archive);
  elements.latestRelease.textContent = latest.version;
  elements.archiveChange.textContent = formatPercent(change);
  elements.archiveChange.className = change >= 0 ? "delta-positive" : "delta-negative";
  elements.firstRelease.textContent = `from ${first.version}`;
  elements.peakArchive.textContent = formatBytes(peak.archive);
  elements.peakRelease.textContent = peak.version;
  elements.releaseCoverage.textContent = `${rows.length} / ${state.releases.length}`;
  elements.archiveCount.textContent = `${pluralize(rows.length, "archive")} measured`;
}

function updateArchiveChart() {
  destroyChart("archive");
  state.charts.archive = new Chart(document.querySelector("#archiveChart"), {
    type: "line",
    data: {
      labels: state.rows.map((row) => shortVersion(row.version)),
      datasets: [
        lineDataset("Distribution archive", state.rows.map((row) => row.platform ? row.archive : null), COLORS.cyan, COLORS.cyanFill),
        lineDataset("Executable payload", state.rows.map((row) => row.platform ? row.executablePayload : null), COLORS.coral, COLORS.coralFill),
      ],
    },
    options: commonChartOptions((context) => formatTooltip(context.dataset.label, context.parsed.y), selectReleaseFromChart),
  });
}

function updateCompositionChart() {
  destroyChart("composition");
  state.charts.composition = new Chart(document.querySelector("#compositionChart"), {
    type: "bar",
    data: {
      labels: state.rows.map((row) => shortVersion(row.version)),
      datasets: [
        barDataset("Commands", state.rows.map((row) => row.platform ? row.commands : null), COLORS.yellow),
        barDataset("Internal tools", state.rows.map((row) => row.platform ? row.internalTools : null), COLORS.green),
      ],
    },
    options: commonChartOptions((context) => formatTooltip(context.dataset.label, context.parsed.y), selectReleaseFromChart, true),
  });
}

function updateToolChart() {
  const data = state.rows.map((row) => toolForRow(row, state.tool)?.size ?? null);
  const coverage = data.filter((value) => value !== null).length;
  elements.toolChartTitle.textContent = `${state.tool} binary`;
  elements.toolCoverage.textContent = `${coverage}/${state.releases.length} releases`;
  elements.toolColumnHeader.textContent = state.tool;

  destroyChart("tool");
  state.charts.tool = new Chart(document.querySelector("#toolChart"), {
    type: "line",
    data: {
      labels: state.rows.map((row) => shortVersion(row.version)),
      datasets: [lineDataset(state.tool, data, COLORS.coral, COLORS.coralFill)],
    },
    options: commonChartOptions((context) => formatTooltip(state.tool, context.parsed.y), selectReleaseFromChart),
  });
}

function updateSnapshot() {
  const row = state.rows.find((candidate) => candidate.version === state.release && candidate.platform);
  if (!row) {
    return;
  }

  elements.snapshotTitle.textContent = `${row.version} snapshot`;
  elements.snapshotTotal.textContent = formatBytes(row.executablePayload);
  updateSnapshotChart(row);
  updateTopTools(row);
}

function updateSnapshotChart(row) {
  destroyChart("snapshot");
  state.charts.snapshot = new Chart(document.querySelector("#snapshotChart"), {
    type: "doughnut",
    data: {
      labels: ["Commands", "Internal tools"],
      datasets: [{
        data: [row.commands, row.internalTools],
        backgroundColor: [COLORS.yellow, COLORS.green],
        borderColor: "#ffffff",
        borderWidth: 3,
        hoverOffset: 4,
      }],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      cutout: "66%",
      plugins: {
        legend: { position: "bottom" },
        tooltip: { callbacks: { label: (context) => formatTooltip(context.label, context.parsed) } },
      },
    },
  });
}

function updateTopTools(row) {
  const tools = [...row.platform.tools].sort((left, right) => right.size - left.size).slice(0, 5);
  const largest = tools[0]?.size || 1;
  elements.topTools.replaceChildren(...tools.map((tool) => {
    const item = document.createElement("li");
    const name = document.createElement("span");
    name.className = "tool-name";
    name.textContent = tool.name;
    const size = document.createElement("span");
    size.className = "tool-size";
    size.textContent = formatBytes(tool.size);
    const bar = document.createElement("span");
    bar.className = "tool-bar";
    const fill = document.createElement("span");
    fill.style.width = `${Math.max(3, (tool.size / largest) * 100)}%`;
    bar.append(fill);
    item.append(name, size, bar);
    return item;
  }));
}

function updateTable() {
  const rows = state.rows.filter((row) => row.platform);
  elements.tableCount.textContent = pluralize(rows.length, "release");
  elements.releaseTable.replaceChildren(...[...rows].reverse().map((row) => {
    const chronologicalIndex = rows.indexOf(row);
    const previous = chronologicalIndex > 0 ? rows[chronologicalIndex - 1] : null;
    const change = previous ? ((row.archive / previous.archive) - 1) * 100 : null;
    const selectedTool = toolForRow(row, state.tool);

    const tableRow = document.createElement("tr");
    if (row.version === state.release) {
      tableRow.setAttribute("aria-current", "true");
    }
    tableRow.addEventListener("click", () => setRelease(row.version));
    tableRow.append(
      cell(row.version, "release-cell"),
      cell(formatBytes(row.archive)),
      cell(change === null ? "—" : formatPercent(change), change === null ? "" : change >= 0 ? "delta-positive" : "delta-negative"),
      cell(formatBytes(row.executablePayload)),
      cell(selectedTool ? formatBytes(selectedTool.size) : "—"),
      cell(String(row.toolCount)),
      cell(row.platform.archive.filename, "artifact-cell", row.platform.archive.filename),
    );
    return tableRow;
  }));
}

function cell(text, className = "", title = "") {
  const tableCell = document.createElement("td");
  tableCell.textContent = text;
  tableCell.className = className;
  if (title) {
    tableCell.title = title;
  }
  return tableCell;
}

function selectReleaseFromChart(_event, activeElements) {
  if (activeElements.length > 0) {
    const row = state.rows[activeElements[0].index];
    if (row?.platform) {
      setRelease(row.version);
    }
  }
}

function setRelease(version) {
  state.release = version;
  elements.releaseSelect.value = version;
  updateSnapshot();
  updateTable();
}

function commonChartOptions(tooltipLabel, onClick, stacked = false) {
  return {
    responsive: true,
    maintainAspectRatio: false,
    normalized: true,
    interaction: { mode: "index", intersect: false },
    onClick,
    scales: {
      x: {
        stacked,
        grid: { display: false },
        ticks: { maxRotation: 0, autoSkip: true, maxTicksLimit: 14 },
      },
      y: {
        stacked,
        beginAtZero: true,
        grid: { color: "rgba(82, 97, 104, 0.12)" },
        border: { display: false },
        ticks: { callback: (value) => `${Math.round(value / MEBIBYTE)} MiB` },
      },
    },
    plugins: {
      legend: { position: "bottom", align: "start" },
      tooltip: { callbacks: { label: tooltipLabel } },
    },
  };
}

function lineDataset(label, data, borderColor, backgroundColor) {
  return {
    label,
    data,
    borderColor,
    backgroundColor,
    borderWidth: 2,
    pointBackgroundColor: borderColor,
    pointBorderColor: "#ffffff",
    pointBorderWidth: 1,
    pointRadius: 3,
    pointHoverRadius: 5,
    tension: 0.22,
    spanGaps: false,
    fill: true,
  };
}

function barDataset(label, data, backgroundColor) {
  return {
    label,
    data,
    backgroundColor,
    borderColor: backgroundColor,
    borderWidth: 0,
    borderRadius: 2,
    maxBarThickness: 24,
  };
}

function destroyChart(name) {
  state.charts[name]?.destroy();
}

function preferredPlatform(release, key) {
  const candidates = release.platforms.filter((platform) => platformKey(platform) === key);
  if (candidates.length < 2) {
    return candidates[0] || null;
  }
  return [...candidates].sort((left, right) => archivePreference(left) - archivePreference(right) || left.archive.filename.localeCompare(right.archive.filename)).at(-1);
}

function archivePreference(platform) {
  if (platform.archive.filename.includes("osx10.8")) {
    return 2;
  }
  if (platform.archive.filename.includes("osx10.6")) {
    return 1;
  }
  return 0;
}

function toolForRow(row, name) {
  return row.platform?.tools.find((tool) => tool.name === name) || null;
}

function platformKey(platform) {
  return `${platform.os}/${platform.arch}`;
}

function platformLabel(key) {
  const [os, arch] = key.split("/");
  return `${OS_NAMES[os] || os} / ${ARCH_NAMES[arch] || arch}`;
}

function compareReleases(left, right) {
  const leftParts = versionParts(left.version);
  const rightParts = versionParts(right.version);
  for (let index = 0; index < 3; index += 1) {
    if (leftParts[index] !== rightParts[index]) {
      return leftParts[index] - rightParts[index];
    }
  }
  return 0;
}

function versionParts(version) {
  const match = version.match(/^go(\d+)\.(\d+)(?:\.(\d+))?/);
  return match ? [Number(match[1]), Number(match[2]), Number(match[3] || 0)] : [0, 0, 0];
}

function shortVersion(version) {
  return version.replace(/^go/, "");
}

function updateDatasetMeta() {
  const first = state.releases[0].version;
  const latest = state.releases.at(-1).version;
  elements.datasetRange.textContent = `${first} → ${latest} · ${state.releases.length} releases`;
  const generated = new Date(state.report.generated_at);
  elements.generatedAt.dateTime = state.report.generated_at;
  elements.generatedAt.textContent = `Generated ${new Intl.DateTimeFormat("en", { dateStyle: "medium", timeZone: "UTC" }).format(generated)} UTC`;
}

function downloadCsv() {
  const [os, arch] = state.platform.split("/");
  const header = ["release", "os", "arch", "archive_filename", "archive_bytes", "executable_payload_bytes", "command_bytes", "internal_tool_bytes", "tool_count"];
  const lines = [header.join(",")];
  for (const row of state.rows.filter((candidate) => candidate.platform)) {
    lines.push([
      row.version,
      os,
      arch,
      row.platform.archive.filename,
      row.archive,
      row.executablePayload,
      row.commands,
      row.internalTools,
      row.toolCount,
    ].map(csvValue).join(","));
  }

  const blob = new Blob([`${lines.join("\n")}\n`], { type: "text/csv;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = `go-size-history-${os}-${arch}.csv`;
  link.hidden = true;
  document.body.append(link);
  link.click();
  link.remove();
  setTimeout(() => URL.revokeObjectURL(url), 0);
}

function csvValue(value) {
  const text = String(value);
  return /[",\n]/.test(text) ? `"${text.replaceAll('"', '""')}"` : text;
}

function formatBytes(bytes) {
  if (!Number.isFinite(bytes)) {
    return "—";
  }
  return `${(bytes / MEBIBYTE).toFixed(bytes >= 100 * MEBIBYTE ? 0 : 1)} MiB`;
}

function formatTooltip(label, bytes) {
  return `${label}: ${formatBytes(bytes)} (${new Intl.NumberFormat("en").format(bytes)} B)`;
}

function formatPercent(value) {
  const sign = value > 0 ? "+" : "";
  return `${sign}${value.toFixed(1)}%`;
}

function pluralize(count, noun) {
  return `${count} ${noun}${count === 1 ? "" : "s"}`;
}

function sum(values) {
  return values.reduce((total, value) => total + value, 0);
}

function showError(error) {
  console.error(error);
  elements.status.className = "status status--error";
  elements.status.textContent = `Unable to load dashboard: ${error.message}`;
}