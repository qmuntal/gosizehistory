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
  latestFootprint: document.querySelector("#latestFootprint"),
  latestRelease: document.querySelector("#latestRelease"),
  footprintChange: document.querySelector("#footprintChange"),
  firstRelease: document.querySelector("#firstRelease"),
  peakFootprint: document.querySelector("#peakFootprint"),
  peakRelease: document.querySelector("#peakRelease"),
  releaseCoverage: document.querySelector("#releaseCoverage"),
  releaseCount: document.querySelector("#releaseCount"),
  snapshotTitle: document.querySelector("#snapshotTitle"),
  snapshotTotal: document.querySelector("#snapshotTotal"),
  toolChartTitle: document.querySelector("#toolChartTitle"),
  toolCoverage: document.querySelector("#toolCoverage"),
  toolColumnHeader: document.querySelector("#toolColumnHeader"),
  releaseTable: document.querySelector("#releaseTable"),
  tableCount: document.querySelector("#tableCount"),
  compareModeButtons: [...document.querySelectorAll("[data-compare-mode]")],
  compareScope: document.querySelector("#compareScope"),
  compareRightLabel: document.querySelector("#compareRightLabel"),
  compareRightSelect: document.querySelector("#compareRightSelect"),
  compareLeftName: document.querySelector("#compareLeftName"),
  compareRightName: document.querySelector("#compareRightName"),
  compareLeftTotal: document.querySelector("#compareLeftTotal"),
  compareRightTotal: document.querySelector("#compareRightTotal"),
  compareLeftCount: document.querySelector("#compareLeftCount"),
  compareRightCount: document.querySelector("#compareRightCount"),
  compareLeftLargest: document.querySelector("#compareLeftLargest"),
  compareRightLargest: document.querySelector("#compareRightLargest"),
  compareLeftAverage: document.querySelector("#compareLeftAverage"),
  compareRightAverage: document.querySelector("#compareRightAverage"),
  compareDelta: document.querySelector("#compareDelta"),
  compareDeltaBytes: document.querySelector("#compareDeltaBytes"),
  compareTableLeft: document.querySelector("#compareTableLeft"),
  compareTableRight: document.querySelector("#compareTableRight"),
  compareTable: document.querySelector("#compareTable"),
};

const state = {
  report: null,
  releases: [],
  platform: "linux/amd64",
  release: "",
  tool: "compile",
  rows: [],
  comparison: {
    mode: "releases",
    context: "",
    left: "",
    right: "",
  },
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

    elements.status.hidden = true;
    elements.metricGrid.hidden = false;
    elements.visuals.hidden = false;
    elements.visuals.getBoundingClientRect();

    updatePlatform();
    configureComparison(true);
    updateDatasetMeta();

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
    setRelease(elements.releaseSelect.value);
  });
  elements.toolSelect.addEventListener("change", () => {
    state.tool = elements.toolSelect.value;
    updateToolChart();
    updateTable();
  });
  elements.downloadCsv.addEventListener("click", downloadCsv);
  for (const button of elements.compareModeButtons) {
    button.addEventListener("click", () => {
      if (state.comparison.mode === button.dataset.compareMode) {
        return;
      }
      state.comparison.mode = button.dataset.compareMode;
      configureComparison(true);
    });
  }
  elements.compareRightSelect.addEventListener("change", () => {
    state.comparison.right = elements.compareRightSelect.value;
    renderComparison();
  });
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

  const keys = [...coverage.keys()].sort(comparePlatformKeys);
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

function updatePlatform(synchronize = true) {
  const selectedRelease = state.release;
  state.rows = state.releases.map((release) => makeRow(release, preferredPlatform(release, state.platform)));
  const availableRows = state.rows.filter((row) => row.platform);
  state.release = availableRows.some((row) => row.version === selectedRelease)
    ? selectedRelease
    : availableRows.at(-1)?.version || "";

  populateReleaseSelect(availableRows);
  populateToolSelect(availableRows);
  updateMetrics(availableRows);
  updateFootprintChart();
  updateCountChart();
  updateToolChart();
  updateSnapshot();
  updateTable();
  elements.trendTarget.textContent = platformLabel(state.platform);
  if (synchronize) {
    synchronizeComparisonFromTop();
  }
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
  const executablePayload = sum(platform.tools.map((tool) => tool.size));
  return {
    version: release.version,
    platform,
    executablePayload,
    toolCount: platform.tools.length,
  };
}

function updateMetrics(rows) {
  const first = rows[0];
  const latest = rows.at(-1);
  const peak = rows.reduce((largest, row) => row.executablePayload > largest.executablePayload ? row : largest, rows[0]);
  const change = ((latest.executablePayload / first.executablePayload) - 1) * 100;

  elements.latestFootprint.textContent = formatBytes(latest.executablePayload);
  elements.latestRelease.textContent = latest.version;
  elements.footprintChange.textContent = formatPercent(change);
  elements.footprintChange.className = change >= 0 ? "delta-positive" : "delta-negative";
  elements.firstRelease.textContent = `from ${first.version}`;
  elements.peakFootprint.textContent = formatBytes(peak.executablePayload);
  elements.peakRelease.textContent = peak.version;
  elements.releaseCoverage.textContent = `${rows.length} / ${state.releases.length}`;
  elements.releaseCount.textContent = `${pluralize(rows.length, "release")} measured`;
}

function updateFootprintChart() {
  destroyChart("footprint");
  state.charts.footprint = new Chart(document.querySelector("#footprintChart"), {
    type: "line",
    data: {
      labels: state.rows.map((row) => shortVersion(row.version)),
      datasets: [
        lineDataset("Total executables", state.rows.map((row) => row.platform ? row.executablePayload : null), COLORS.cyan, COLORS.cyanFill),
      ],
    },
    options: commonChartOptions((context) => formatTooltip(context.dataset.label, context.parsed.y), selectReleaseFromChart),
  });
}

function updateCountChart() {
  const options = commonChartOptions((context) => `${context.parsed.y} binaries`, selectReleaseFromChart);
  options.scales.y.ticks = { precision: 0 };
  options.plugins.legend.display = false;

  destroyChart("count");
  state.charts.count = new Chart(document.querySelector("#countChart"), {
    type: "bar",
    data: {
      labels: state.rows.map((row) => shortVersion(row.version)),
      datasets: [
        barDataset("Binaries", state.rows.map((row) => row.platform ? row.toolCount : null), COLORS.green),
      ],
    },
    options,
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

  elements.snapshotTitle.textContent = `${row.version} largest binaries`;
  elements.snapshotTotal.textContent = formatBytes(row.executablePayload);
  updateSnapshotChart(row);
}

function updateSnapshotChart(row) {
  const tools = [...row.platform.tools].sort((left, right) => right.size - left.size).slice(0, 8);
  destroyChart("snapshot");
  state.charts.snapshot = new Chart(document.querySelector("#snapshotChart"), {
    type: "bar",
    data: {
      labels: tools.map((tool) => tool.name),
      datasets: [{
        label: "Binary size",
        data: tools.map((tool) => tool.size),
        backgroundColor: COLORS.coral,
        borderRadius: 2,
        maxBarThickness: 22,
      }],
    },
    options: {
      indexAxis: "y",
      responsive: true,
      maintainAspectRatio: false,
      scales: {
        x: {
          beginAtZero: true,
          grid: { color: "rgba(82, 97, 104, 0.12)" },
          border: { display: false },
          ticks: { callback: (value) => `${Math.round(value / MEBIBYTE)} MiB` },
        },
        y: { grid: { display: false } },
      },
      plugins: {
        legend: { display: false },
        tooltip: { callbacks: { label: (context) => formatTooltip(context.label, context.parsed.x) } },
      },
    },
  });
}

function configureComparison(reset) {
  const comparingReleases = state.comparison.mode === "releases";
  for (const button of elements.compareModeButtons) {
    button.setAttribute("aria-pressed", String(button.dataset.compareMode === state.comparison.mode));
  }

  elements.compareRightLabel.textContent = comparingReleases ? "Compare with release" : "Compare with platform";
  state.comparison.context = comparingReleases ? state.platform : state.release;
  elements.compareScope.textContent = comparingReleases
    ? platformLabel(state.comparison.context)
    : state.comparison.context;
  elements.compareRightSelect.disabled = false;
  populateComparisonSides(true);
}

function populateComparisonSides(reset) {
  const comparingReleases = state.comparison.mode === "releases";
  let options;
  if (comparingReleases) {
    options = [...state.releases]
      .reverse()
      .filter((release) => preferredPlatform(release, state.comparison.context))
      .map((release) => ({ value: release.version, label: release.version }));
  } else {
    const release = releaseByVersion(state.comparison.context);
    options = uniquePlatformKeys(release)
      .sort(comparePlatformKeys)
      .map((key) => ({ value: key, label: platformLabel(key) }));
  }

  const baseline = comparingReleases ? state.release : state.platform;
  state.comparison.left = options.some((option) => option.value === baseline)
    ? baseline
    : options[0]?.value || "";

  const targets = options.filter((option) => option.value !== state.comparison.left);
  if (reset || !targets.some((option) => option.value === state.comparison.right)) {
    state.comparison.right = defaultComparisonTarget(options, targets, comparingReleases);
  }
  setSelectOptions(elements.compareRightSelect, targets, state.comparison.right);
  renderComparison();
}

function defaultComparisonTarget(options, targets, comparingReleases) {
  if (comparingReleases) {
    const baselineIndex = options.findIndex((option) => option.value === state.comparison.left);
    return options[baselineIndex + 1]?.value || options[baselineIndex - 1]?.value || targets[0]?.value || "";
  }
  return preferredOption(targets, "windows/amd64", "linux/amd64");
}

function preferredOption(options, ...values) {
  for (const value of values) {
    if (options.some((option) => option.value === value)) {
      return value;
    }
  }
  return options[0]?.value || "";
}

function setSelectOptions(select, options, selected) {
  select.replaceChildren(...options.map((item) => {
    const option = document.createElement("option");
    option.value = item.value;
    option.textContent = item.label;
    return option;
  }));
  select.value = selected;
}

function synchronizeComparisonFromTop() {
  if (elements.compareRightSelect.disabled || elements.compareRightSelect.options.length === 0) {
    return;
  }
  state.comparison.context = state.comparison.mode === "releases" ? state.platform : state.release;
  elements.compareScope.textContent = state.comparison.mode === "releases"
    ? platformLabel(state.comparison.context)
    : state.comparison.context;
  populateComparisonSides(false);
}

function selectHasValue(select, value) {
  return [...select.options].some((option) => option.value === value);
}

function renderComparison() {
  const entries = comparisonEntries();
  if (!entries) {
    return;
  }
  const left = summarizePlatform(entries.left.label, entries.left.platform);
  const right = summarizePlatform(entries.right.label, entries.right.platform);
  const delta = right.total - left.total;
  const percent = left.total === 0 ? 0 : (delta / left.total) * 100;

  renderComparisonSide("left", left);
  renderComparisonSide("right", right);
  elements.compareDelta.textContent = formatPercent(percent);
  elements.compareDelta.className = percent >= 0 ? "delta-positive" : "delta-negative";
  elements.compareDeltaBytes.textContent = formatSignedBytes(delta);
  elements.compareTableLeft.textContent = left.label;
  elements.compareTableRight.textContent = right.label;

  const tools = comparisonTools(left, right);
  updateComparisonChart(left, right, tools.slice(0, 12));
  updateComparisonTable(tools);
}

function comparisonEntries() {
  if (state.comparison.mode === "releases") {
    const leftRelease = releaseByVersion(state.comparison.left);
    const rightRelease = releaseByVersion(state.comparison.right);
    const leftPlatform = preferredPlatform(leftRelease, state.comparison.context);
    const rightPlatform = preferredPlatform(rightRelease, state.comparison.context);
    if (!leftPlatform || !rightPlatform) {
      return null;
    }
    return {
      left: { label: leftRelease.version, platform: leftPlatform },
      right: { label: rightRelease.version, platform: rightPlatform },
    };
  }

  const release = releaseByVersion(state.comparison.context);
  const leftPlatform = preferredPlatform(release, state.comparison.left);
  const rightPlatform = preferredPlatform(release, state.comparison.right);
  if (!leftPlatform || !rightPlatform) {
    return null;
  }
  return {
    left: { label: platformLabel(state.comparison.left), platform: leftPlatform },
    right: { label: platformLabel(state.comparison.right), platform: rightPlatform },
  };
}

function summarizePlatform(label, platform) {
  const byTool = new Map(platform.tools.map((tool) => [tool.name, tool.size]));
  const tools = [...platform.tools].sort((left, right) => right.size - left.size);
  const total = sum(tools.map((tool) => tool.size));
  return {
    label,
    total,
    count: tools.length,
    average: tools.length === 0 ? 0 : total / tools.length,
    largest: tools[0] || null,
    byTool,
  };
}

function renderComparisonSide(side, summary) {
  const title = side === "left" ? elements.compareLeftName : elements.compareRightName;
  const total = side === "left" ? elements.compareLeftTotal : elements.compareRightTotal;
  const count = side === "left" ? elements.compareLeftCount : elements.compareRightCount;
  const largest = side === "left" ? elements.compareLeftLargest : elements.compareRightLargest;
  const average = side === "left" ? elements.compareLeftAverage : elements.compareRightAverage;
  title.textContent = summary.label;
  total.textContent = formatBytes(summary.total);
  count.textContent = String(summary.count);
  largest.textContent = summary.largest ? `${summary.largest.name} · ${formatBytes(summary.largest.size)}` : "—";
  average.textContent = formatBytes(summary.average);
}

function comparisonTools(left, right) {
  const names = new Set([...left.byTool.keys(), ...right.byTool.keys()]);
  return [...names].map((name) => ({
    name,
    left: left.byTool.get(name) ?? null,
    right: right.byTool.get(name) ?? null,
  })).sort((first, second) => Math.max(second.left || 0, second.right || 0) - Math.max(first.left || 0, first.right || 0));
}

function updateComparisonChart(left, right, tools) {
  destroyChart("comparison");
  state.charts.comparison = new Chart(document.querySelector("#comparisonChart"), {
    type: "bar",
    data: {
      labels: tools.map((tool) => tool.name),
      datasets: [
        barDataset(left.label, tools.map((tool) => tool.left || 0), COLORS.cyan),
        barDataset(right.label, tools.map((tool) => tool.right || 0), COLORS.coral),
      ],
    },
    options: {
      indexAxis: "y",
      responsive: true,
      maintainAspectRatio: false,
      interaction: { mode: "index", intersect: false },
      scales: {
        x: {
          beginAtZero: true,
          grid: { color: "rgba(82, 97, 104, 0.12)" },
          border: { display: false },
          ticks: { callback: (value) => `${Math.round(value / MEBIBYTE)} MiB` },
        },
        y: { grid: { display: false } },
      },
      plugins: {
        legend: { position: "bottom", align: "start" },
        tooltip: { callbacks: { label: (context) => formatTooltip(context.dataset.label, context.parsed.x) } },
      },
    },
  });
}

function updateComparisonTable(tools) {
  elements.compareTable.replaceChildren(...tools.map((tool) => {
    const row = document.createElement("tr");
    let delta = "—";
    let deltaClass = "";
    if (tool.left === null) {
      delta = "New";
      deltaClass = "delta-positive";
    } else if (tool.right === null) {
      delta = "Removed";
      deltaClass = "delta-negative";
    } else if (tool.left > 0) {
      const percent = ((tool.right / tool.left) - 1) * 100;
      delta = formatPercent(percent);
      deltaClass = percent >= 0 ? "delta-positive" : "delta-negative";
    }
    row.append(
      cell(tool.name, "release-cell"),
      cell(tool.left === null ? "—" : formatBytes(tool.left)),
      cell(tool.right === null ? "—" : formatBytes(tool.right)),
      cell(delta, deltaClass),
    );
    return row;
  }));
}

function updateTable() {
  const rows = state.rows.filter((row) => row.platform);
  elements.tableCount.textContent = pluralize(rows.length, "release");
  elements.releaseTable.replaceChildren(...[...rows].reverse().map((row) => {
    const chronologicalIndex = rows.indexOf(row);
    const previous = chronologicalIndex > 0 ? rows[chronologicalIndex - 1] : null;
    const change = previous ? ((row.executablePayload / previous.executablePayload) - 1) * 100 : null;
    const selectedTool = toolForRow(row, state.tool);
    const largest = [...row.platform.tools].sort((left, right) => right.size - left.size)[0];

    const tableRow = document.createElement("tr");
    if (row.version === state.release) {
      tableRow.setAttribute("aria-current", "true");
    }
    tableRow.addEventListener("click", () => setRelease(row.version));
    tableRow.append(
      cell(row.version, "release-cell"),
      cell(formatBytes(row.executablePayload)),
      cell(change === null ? "—" : formatPercent(change), change === null ? "" : change >= 0 ? "delta-positive" : "delta-negative"),
      cell(selectedTool ? formatBytes(selectedTool.size) : "—"),
      cell(String(row.toolCount)),
      cell(largest ? `${largest.name} · ${formatBytes(largest.size)}` : "—", "binary-cell"),
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

function setRelease(version, synchronize = true) {
  state.release = version;
  elements.releaseSelect.value = version;
  updateSnapshot();
  updateTable();
  if (synchronize) {
    synchronizeComparisonFromTop();
  }
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
  if (!release) {
    return null;
  }
  const candidates = release.platforms.filter((platform) => platformKey(platform) === key);
  if (candidates.length < 2) {
    return candidates[0] || null;
  }
  return [...candidates].sort((left, right) => archivePreference(left) - archivePreference(right) || left.archive.filename.localeCompare(right.archive.filename)).at(-1);
}

function releaseByVersion(version) {
  return state.releases.find((release) => release.version === version) || null;
}

function uniquePlatformKeys(release) {
  if (!release) {
    return [];
  }
  return [...new Set(release.platforms.map(platformKey))];
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
  const header = ["release", "os", "arch", "executable_payload_bytes", "binary_count", "largest_binary", "largest_binary_bytes"];
  const lines = [header.join(",")];
  for (const row of state.rows.filter((candidate) => candidate.platform)) {
    const largest = [...row.platform.tools].sort((left, right) => right.size - left.size)[0];
    lines.push([
      row.version,
      os,
      arch,
      row.executablePayload,
      row.toolCount,
      largest?.name || "",
      largest?.size || 0,
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

function formatSignedBytes(bytes) {
  if (bytes === 0) {
    return "0 MiB";
  }
  return `${bytes > 0 ? "+" : "−"}${formatBytes(Math.abs(bytes))}`;
}

function pluralize(count, noun) {
  return `${count} ${noun}${count === 1 ? "" : "s"}`;
}

function sum(values) {
  return values.reduce((total, value) => total + value, 0);
}

function showError(error) {
  console.error(error);
  elements.status.hidden = false;
  elements.status.className = "status status--error";
  elements.status.textContent = `Unable to load dashboard: ${error.message}`;
}

function comparePlatformKeys(left, right) {
  return platformLabel(left).localeCompare(platformLabel(right), "en", {
    numeric: true,
    sensitivity: "base",
  });
}