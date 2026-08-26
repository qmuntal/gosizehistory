const DATA_FILENAME = "go-tool-sizes.json";
const DATA_VERSION = "20260826-root-report";
const DATA_URL = `${location.pathname.includes("/dashboard/") ? "../" : "./"}${DATA_FILENAME}?v=${DATA_VERSION}`;
const MEBIBYTE = 1024 * 1024;

const COLOR_SCHEMES = {
  light: {
    ink: "#20343a",
    inkSoft: "#5b6b6f",
    line: "#d0ddda",
    gopher: "#00add8",
    gopherFill: "rgba(0, 173, 216, 0.14)",
    amber: "#d48218",
    amberFill: "rgba(212, 130, 24, 0.13)",
    graphite: "#40575e",
    teal: "#2b7f74",
    grid: "rgba(32, 52, 58, 0.11)",
    surface: "#fcfdfd",
  },
  dark: {
    ink: "#e4eeee",
    inkSoft: "#a8b8b6",
    line: "#2c4145",
    gopher: "#00add8",
    gopherFill: "rgba(0, 173, 216, 0.18)",
    amber: "#e3a044",
    amberFill: "rgba(227, 160, 68, 0.16)",
    graphite: "#789197",
    teal: "#52a596",
    grid: "rgba(168, 184, 182, 0.16)",
    surface: "#172429",
  },
};
const COLORS = {};

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
  downloadJson: document.querySelector("#downloadJson"),
  tipNotice: document.querySelector("#tipNotice"),
  tipVersion: document.querySelector("#tipVersion"),
  tipRevision: document.querySelector("#tipRevision"),
  tipPlatforms: document.querySelector("#tipPlatforms"),
  tipCommitTime: document.querySelector("#tipCommitTime"),
  tipSource: document.querySelector("#tipSource"),
  generatedAt: document.querySelector("#generatedAt"),
  datasetRange: document.querySelector("#datasetRange"),
  trendTarget: document.querySelector("#trendTarget"),
  trendChartTitle: document.querySelector("#trendChartTitle"),
  trendNote: document.querySelector("#trendNote"),
  toolNote: document.querySelector("#toolNote"),
  currentFootprint: document.querySelector("#currentFootprint"),
  currentFootprintContext: document.querySelector("#currentFootprintContext"),
  recentSizeChangeLabel: document.querySelector("#recentSizeChangeLabel"),
  recentSizeChange: document.querySelector("#recentSizeChange"),
  recentSizeChangeContext: document.querySelector("#recentSizeChangeContext"),
  historySizeChange: document.querySelector("#historySizeChange"),
  historySizeChangeContext: document.querySelector("#historySizeChangeContext"),
  releaseCoverage: document.querySelector("#releaseCoverage"),
  releaseCount: document.querySelector("#releaseCount"),
  snapshotTitle: document.querySelector("#snapshotTitle"),
  snapshotTotal: document.querySelector("#snapshotTotal"),
  snapshotInsight: document.querySelector("#snapshotInsight"),
  toolChartTitle: document.querySelector("#toolChartTitle"),
  toolCoverage: document.querySelector("#toolCoverage"),
  toolColumnHeader: document.querySelector("#toolColumnHeader"),
  releaseTable: document.querySelector("#releaseTable"),
  tableCount: document.querySelector("#tableCount"),
  compareModeButtons: [...document.querySelectorAll("[data-compare-mode]")],
  themeToggle: document.querySelector("#themeToggle"),
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
  compareDeltaSummary: document.querySelector("#compareDeltaSummary"),
  compareDeltaLabel: document.querySelector("#compareDeltaLabel"),
  compareDeltaHeader: document.querySelector("#compareDeltaHeader"),
  compareTableLeft: document.querySelector("#compareTableLeft"),
  compareTableRight: document.querySelector("#compareTableRight"),
  compareTable: document.querySelector("#compareTable"),
  comparisonAxisNote: document.querySelector("#comparisonAxisNote"),
  comparisonInsight: document.querySelector("#comparisonInsight"),
  trendModeButtons: [...document.querySelectorAll("[data-trend-mode]")],
  heatmapModeButtons: [...document.querySelectorAll("[data-heatmap-mode]")],
  heatmap: document.querySelector("#heatmap"),
  heatmapNote: document.querySelector("#heatmapNote"),
  heatmapLegend: document.querySelector("#heatmapLegend"),
};

const state = {
  report: null,
  tipRelease: null,
  releases: [],
  platform: "linux/amd64",
  release: "",
  tool: "compile",
  trendMode: "absolute",
  heatmapMode: "absolute",
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
  syncThemeToggle();

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
    elements.downloadJson.href = DATA_URL.split("?")[0];
    state.tipRelease = state.report.releases.find((release) => release.development) || null;
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
    updateTipNotice();

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
  applyChartTheme();
  Chart.defaults.color = COLORS.inkSoft;
  Chart.defaults.borderColor = COLORS.line;
  Chart.defaults.font.family = "IBM Plex Sans, sans-serif";
  Chart.defaults.font.size = 11;
  Chart.defaults.animation.duration = window.matchMedia("(prefers-reduced-motion: reduce)").matches ? 0 : 450;
  Chart.defaults.plugins.legend.labels.boxWidth = 12;
  Chart.defaults.plugins.legend.labels.boxHeight = 3;
  Chart.defaults.plugins.legend.labels.usePointStyle = true;
  Chart.defaults.plugins.legend.labels.pointStyle = "line";
}

function bindControls() {
  elements.themeToggle.addEventListener("click", toggleTheme);
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
  for (const button of elements.trendModeButtons) {
    button.addEventListener("click", () => setTrendMode(button.dataset.trendMode));
  }
  for (const button of elements.heatmapModeButtons) {
    button.addEventListener("click", () => setHeatmapMode(button.dataset.heatmapMode));
  }
  elements.compareRightSelect.addEventListener("change", () => {
    state.comparison.right = elements.compareRightSelect.value;
    renderComparison();
  });
}

function currentTheme() {
  return document.documentElement.dataset.theme === "dark" ? "dark" : "light";
}

function toggleTheme() {
  const theme = currentTheme() === "dark" ? "light" : "dark";
  document.documentElement.dataset.theme = theme;
  try {
    localStorage.setItem("gosizehistory-theme", theme);
  } catch {
    // The selected theme still applies for this page when storage is unavailable.
  }
  syncThemeToggle();
  applyChartTheme();
  updatePlatform();
}

function syncThemeToggle() {
  const dark = currentTheme() === "dark";
  const label = dark ? "Use light theme" : "Use dark theme";
  elements.themeToggle.setAttribute("aria-pressed", String(dark));
  elements.themeToggle.setAttribute("aria-label", label);
  elements.themeToggle.title = label;
}

function applyChartTheme() {
  Object.assign(COLORS, COLOR_SCHEMES[currentTheme()]);
  if (window.Chart) {
    Chart.defaults.color = COLORS.inkSoft;
    Chart.defaults.borderColor = COLORS.line;
  }
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
    option.textContent = platformLabel(key);
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
  updateHeatmap();
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

  const names = [...coverage.keys()].sort(compareAlphaNumeric);
  elements.toolSelect.replaceChildren(...names.map((name) => {
    const option = document.createElement("option");
    option.value = name;
    option.textContent = name;
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
  const stableRows = rows.filter((row) => releaseByVersion(row.version)?.stable);
  const tipRow = rows.find((row) => releaseByVersion(row.version)?.development);
  const latestStable = stableRows.at(-1);
  const current = rows.at(-1);
  const first = rows[0];
  const previous = rows.at(-2);

  if (current) {
    elements.currentFootprint.textContent = formatBytes(current.executablePayload);
    elements.currentFootprint.className = "";
    elements.currentFootprintContext.textContent = `${current.version} · ${pluralize(current.toolCount, "binary", "binaries")}`;
  } else {
    setUnavailableMetric(elements.currentFootprint, elements.currentFootprintContext, "No measurements");
  }

  const recentFrom = tipRow && latestStable ? latestStable : previous;
  const recentTo = tipRow && latestStable ? tipRow : current;
  elements.recentSizeChangeLabel.textContent = tipRow && latestStable ? "Tip vs latest stable" : "Latest measured change";
  if (recentFrom && recentTo && recentFrom !== recentTo) {
    setSizeChangeMetric(
      elements.recentSizeChange,
      elements.recentSizeChangeContext,
      sizeChangePercent(recentFrom.executablePayload, recentTo.executablePayload),
      `${recentFrom.version} → ${recentTo.version}`,
    );
  } else {
    setUnavailableMetric(elements.recentSizeChange, elements.recentSizeChangeContext, "Insufficient history");
  }

  if (first && current && first !== current) {
    setSizeChangeMetric(
      elements.historySizeChange,
      elements.historySizeChangeContext,
      sizeChangePercent(first.executablePayload, current.executablePayload),
      `${first.version} → ${current.version}`,
    );
  } else {
    setUnavailableMetric(elements.historySizeChange, elements.historySizeChangeContext, "Insufficient history");
  }
  elements.releaseCoverage.textContent = `${rows.length} / ${state.releases.length}`;
  elements.releaseCount.textContent = `${formatUnsignedPercent((rows.length / state.releases.length) * 100)} measured`;
}

function setSizeChangeMetric(valueElement, contextElement, percent, context) {
  valueElement.textContent = formatPercent(percent);
  valueElement.className = percent > 0 ? "delta-regression" : "delta-improvement";
  contextElement.textContent = `${sizeChangeDirection(percent)} · ${context}`;
}

function sizeChangeDirection(percent) {
  return percent < 0 ? "smaller" : percent > 0 ? "larger" : "unchanged";
}

function setUnavailableMetric(valueElement, contextElement, context) {
  valueElement.textContent = "—";
  valueElement.className = "";
  contextElement.textContent = context;
}

function sizeChangePercent(earlier, later) {
  return earlier === 0 ? 0 : ((later - earlier) / earlier) * 100;
}

function setTrendMode(mode) {
  if (!mode || state.trendMode === mode) {
    return;
  }
  state.trendMode = mode;
  updateModeButtons(elements.trendModeButtons, mode, "trendMode");
  updateFootprintChart();
  updateToolChart();
}

function setHeatmapMode(mode) {
  if (!mode || state.heatmapMode === mode) {
    return;
  }
  state.heatmapMode = mode;
  updateModeButtons(elements.heatmapModeButtons, mode, "heatmapMode");
  updateHeatmap();
}

function updateModeButtons(buttons, mode, dataKey) {
  for (const button of buttons) {
    button.setAttribute("aria-pressed", String(button.dataset[dataKey] === mode));
  }
}

function updateFootprintChart() {
  const series = transformTrendSeries(state.rows, (row) => row.platform ? row.executablePayload : null);
  const mode = trendModeDefinition();
  elements.trendChartTitle.textContent = mode.toolchainTitle;
  elements.trendNote.textContent = trendNote(series, "Gaps indicate an unavailable platform.");

  destroyChart("footprint");
  state.charts.footprint = new Chart(labeledChartCanvas(
    "#footprintChart",
    `${mode.toolchainTitle} for ${platformLabel(state.platform)}`,
  ), {
    type: "line",
    data: {
      labels: state.rows.map((row) => shortVersion(row.version)),
      datasets: [
        lineDataset(mode.datasetLabel, series.values, COLORS.gopher, COLORS.gopherFill, mode.fill),
      ],
    },
    options: commonChartOptions(
      trendTooltipCallbacks(state.rows, (row) => row.platform ? row.executablePayload : null, "Footprint"),
      selectReleaseFromChart,
      false,
      mode.axisKind,
    ),
  });
}

function updateCountChart() {
  const options = commonChartOptions({
    title: (items) => state.rows[items[0]?.dataIndex]?.version || "",
    label: (context) => `${context.parsed.y} binaries`,
    afterLabel: (context) => {
      const index = context.dataIndex;
      if (index === 0 || !state.rows[index - 1].platform) {
        return "No prior comparable release";
      }
      const difference = context.parsed.y - state.rows[index - 1].toolCount;
      return `Vs prior: ${difference > 0 ? "+" : ""}${difference}`;
    },
  }, selectReleaseFromChart, false, "count");
  options.plugins.legend.display = false;

  destroyChart("count");
  state.charts.count = new Chart(labeledChartCanvas(
    "#countChart",
    `Shipped binary count by release for ${platformLabel(state.platform)}`,
  ), {
    type: "bar",
    data: {
      labels: state.rows.map((row) => shortVersion(row.version)),
      datasets: [
        barDataset("Binaries", state.rows.map((row) => row.platform ? row.toolCount : null), COLORS.teal),
      ],
    },
    options,
  });
}

function updateToolChart() {
  const rawValues = state.rows.map((row) => toolForRow(row, state.tool)?.size ?? null);
  const series = transformTrendSeries(state.rows, (_row, index) => rawValues[index]);
  const coverage = rawValues.filter((value) => value !== null).length;
  const mode = trendModeDefinition();
  elements.toolChartTitle.textContent = `${state.tool} · ${mode.binaryTitle}`;
  elements.toolCoverage.textContent = `${coverage}/${state.releases.length} releases · ${state.releases.length - coverage} gaps`;
  elements.toolNote.textContent = trendNote(series, "Gaps indicate an unavailable platform or binary.");
  elements.toolColumnHeader.textContent = `${state.tool} size`;

  destroyChart("tool");
  state.charts.tool = new Chart(labeledChartCanvas(
    "#toolChart",
    `${state.tool} ${mode.binaryTitle} for ${platformLabel(state.platform)}`,
  ), {
    type: "line",
    data: {
      labels: state.rows.map((row) => shortVersion(row.version)),
      datasets: [lineDataset(state.tool, series.values, COLORS.amber, COLORS.amberFill, mode.fill)],
    },
    options: commonChartOptions(
      trendTooltipCallbacks(state.rows, (row) => toolForRow(row, state.tool)?.size ?? null, state.tool, true),
      selectReleaseFromChart,
      false,
      mode.axisKind,
    ),
  });
}

function transformTrendSeries(rows, valueForRow) {
  const raw = rows.map((row, index) => valueForRow(row, index));
  if (state.trendMode === "absolute") {
    return { values: raw, baseline: null };
  }
  if (state.trendMode === "indexed") {
    const baselineIndex = raw.findIndex((value) => value !== null);
    if (baselineIndex < 0) {
      return { values: raw, baseline: null };
    }
    const baseline = raw[baselineIndex];
    return {
      values: raw.map((value) => value === null ? null : (value / baseline) * 100),
      baseline: rows[baselineIndex].version,
    };
  }
  return {
    values: raw.map((value, index) => {
      if (index === 0 || value === null || raw[index - 1] === null) {
        return null;
      }
      return sizeChangePercent(raw[index - 1], value);
    }),
    baseline: null,
  };
}

function trendModeDefinition() {
  switch (state.trendMode) {
  case "indexed":
    return {
      axisKind: "index",
      datasetLabel: "Relative footprint",
      toolchainTitle: "Relative footprint (first = 100)",
      binaryTitle: "relative size (first = 100)",
      fill: false,
    };
  case "delta":
    return {
      axisKind: "percent",
      datasetLabel: "Size change vs prior",
      toolchainTitle: "Size change vs prior release",
      binaryTitle: "change vs prior release",
      fill: false,
    };
  default:
    return {
      axisKind: "bytes",
      datasetLabel: "Executable footprint",
      toolchainTitle: "Executable footprint over time",
      binaryTitle: "size over time",
      fill: true,
    };
  }
}

function trendNote(series, missingText) {
  if (state.trendMode === "indexed" && series.baseline) {
    return `${missingText} Baseline: ${series.baseline} = 100.`;
  }
  if (state.trendMode === "delta") {
    return `${missingText} Negative values mean the later release is smaller.`;
  }
  return missingText;
}

function trendTooltipCallbacks(rows, valueForRow, label, includeRank = false) {
  return {
    title: (items) => rows[items[0]?.dataIndex]?.version || "",
    label: (context) => `${label}: ${formatTrendValue(context.parsed.y)}`,
    afterLabel: (context) => {
      const index = context.dataIndex;
      const row = rows[index];
      const value = valueForRow(row, index);
      if (value === null) {
        return "Unavailable";
      }
      const details = [formatExactBytes(value)];
      if (index > 0) {
        const previous = valueForRow(rows[index - 1], index - 1);
        if (previous !== null) {
          const change = sizeChangePercent(previous, value);
          const direction = change < 0 ? "smaller" : change > 0 ? "larger" : "unchanged";
          details.push(`Vs prior: ${formatPercent(change)} · ${direction}`);
        }
      }
      if (row.platform) {
        if (includeRank) {
          const ranked = [...row.platform.tools].sort((left, right) => right.size - left.size);
          const rank = ranked.findIndex((tool) => tool.name === state.tool) + 1;
          details.push(`Share: ${formatUnsignedPercent((value / row.executablePayload) * 100)} · rank ${rank}/${ranked.length}`);
        } else {
          details.push(`${pluralize(row.toolCount, "binary", "binaries")} shipped`);
        }
      }
      const release = releaseByVersion(row.version);
      if (release?.development && release.revision) {
        details.push(`Tip: ${release.revision.slice(0, 12)}`);
      }
      return details;
    },
  };
}

function formatTrendValue(value) {
  switch (state.trendMode) {
  case "indexed":
    return value.toFixed(1);
  case "delta":
    return formatPercent(value);
  default:
    return formatBytes(value);
  }
}

function formatExactBytes(bytes) {
  return `${new Intl.NumberFormat("en").format(bytes)} bytes`;
}

function formatUnsignedPercent(value) {
  return `${value.toFixed(1)}%`;
}

function updateSnapshot() {
  const row = state.rows.find((candidate) => candidate.version === state.release && candidate.platform);
  if (!row) {
    return;
  }

  elements.snapshotTitle.textContent = `${Math.min(8, row.toolCount)} largest binaries in ${row.version}`;
  elements.snapshotTotal.textContent = `${formatBytes(row.executablePayload)} total`;
  const ranked = [...row.platform.tools].sort((left, right) => right.size - left.size);
  const largest = ranked[0];
  const topCount = Math.min(3, ranked.length);
  const topShare = sum(ranked.slice(0, topCount).map((tool) => tool.size)) / row.executablePayload * 100;
  elements.snapshotInsight.textContent = largest
    ? `${largest.name} accounts for ${formatUnsignedPercent(largest.size / row.executablePayload * 100)} of total; top ${topCount} ${topCount === 1 ? "accounts" : "account"} for ${formatUnsignedPercent(topShare)}.`
    : "No binaries measured.";
  updateSnapshotChart(row);
}

function updateSnapshotChart(row) {
  const tools = [...row.platform.tools].sort((left, right) => right.size - left.size).slice(0, 8);
  const release = releaseByVersion(row.version);
  destroyChart("snapshot");
  state.charts.snapshot = new Chart(labeledChartCanvas(
    "#snapshotChart",
    `Largest binaries in ${row.version} for ${platformLabel(state.platform)}`,
  ), {
    type: "bar",
    data: {
      labels: tools.map((tool) => tool.name),
      datasets: [{
        label: "Binary size",
        data: tools.map((tool) => tool.size),
        backgroundColor: COLORS.amber,
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
          ...byteAxisOptions(true),
        },
        y: { grid: { display: false } },
      },
      plugins: {
        legend: { display: false },
        tooltip: {
          callbacks: {
            title: () => row.version,
            label: (context) => `${context.label}: ${formatBytes(context.parsed.x)}`,
            afterLabel: (context) => {
              const tool = tools[context.dataIndex];
              const details = [
                formatExactBytes(tool.size),
                `Share: ${formatUnsignedPercent((tool.size / row.executablePayload) * 100)} · rank ${context.dataIndex + 1}/${row.platform.tools.length}`,
              ];
              if (release?.development && release.revision) {
                details.push(`Tip: ${release.revision.slice(0, 12)}`);
              }
              return details;
            },
          },
        },
      },
      layout: { padding: { right: 8 } },
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
  const change = comparisonChange(left, right);

  renderComparisonSide("left", left);
  renderComparisonSide("right", right);
  elements.compareDeltaLabel.textContent = change.label;
  elements.compareDeltaHeader.textContent = change.columnLabel;
  elements.compareDeltaSummary.setAttribute("aria-label", change.description);
  elements.compareDeltaSummary.title = change.description;
  elements.compareDelta.textContent = formatPercent(change.percent);
  elements.compareDelta.className = change.regression ? "delta-regression" : "delta-improvement";
  elements.compareDeltaBytes.textContent = formatSignedBytes(change.bytes);
  elements.compareTableLeft.textContent = left.label;
  elements.compareTableRight.textContent = right.label;

  const tools = comparisonTools(left, right);
  const analysis = comparisonAnalysis(left, right, tools, change);
  updateComparisonInsight(analysis);
  updateComparisonChart(analysis);
  updateComparisonTable(analysis.contributions, change.earlierSide);
}

function comparisonAnalysis(left, right, tools, change) {
  const releaseMode = state.comparison.mode === "releases";
  const from = releaseMode && change.earlierSide === "right" ? right : left;
  const to = releaseMode && change.earlierSide === "right" ? left : right;
  const contributions = tools.map((tool) => {
    const fromValue = from.byTool.get(tool.name) || 0;
    const toValue = to.byTool.get(tool.name) || 0;
    const rawDelta = toValue - fromValue;
    return {
      ...tool,
      from: fromValue,
      to: toValue,
      rawDelta,
      contribution: rawDelta,
    };
  }).sort((first, second) => Math.abs(second.contribution) - Math.abs(first.contribution));
  return { releaseMode, from, to, change, contributions };
}

function updateComparisonInsight(analysis) {
  const shared = analysis.contributions.filter((tool) => tool.from > 0 && tool.to > 0);
  const smaller = shared.filter((tool) => tool.rawDelta < 0).length;
  const larger = shared.filter((tool) => tool.rawDelta > 0).length;
  const unchanged = shared.length - smaller - larger;
  const added = analysis.contributions.filter((tool) => tool.from === 0 && tool.to > 0).length;
  const removed = analysis.contributions.filter((tool) => tool.from > 0 && tool.to === 0).length;
  const sizeSummary = [
    smaller > 0 ? `${smaller} smaller` : "",
    larger > 0 ? `${larger} larger` : "",
    unchanged > 0 ? `${unchanged} unchanged` : "",
  ].filter(Boolean).join(" · ") || "no shared binaries";
  const inventorySummary = added === 0 && removed === 0
    ? "No binaries added or removed."
    : `${[added > 0 ? `${added} added` : "", removed > 0 ? `${removed} removed` : ""].filter(Boolean).join(" · ")}.`;
  const dominant = analysis.contributions.find((tool) => tool.rawDelta !== 0);
  let dominantSummary = "No size change.";
  if (dominant?.from === 0) {
    dominantSummary = `Largest change: ${dominant.name} added at ${formatBytes(dominant.to)}.`;
  } else if (dominant?.to === 0) {
    dominantSummary = `Largest change: ${dominant.name} removed at ${formatBytes(dominant.from)}.`;
  } else if (dominant) {
    dominantSummary = `Largest change: ${dominant.name} ${formatSignedMiB(dominant.rawDelta / MEBIBYTE)}.`;
  }
  elements.comparisonInsight.textContent = `In ${analysis.to.label}: ${sizeSummary}. ${inventorySummary} ${dominantSummary}`;
}

function comparisonChange(left, right) {
  if (state.comparison.mode === "releases") {
    const leftRelease = releaseByVersion(left.label);
    const rightRelease = releaseByVersion(right.label);
    const earlierSide = compareReleases(leftRelease, rightRelease) <= 0 ? "left" : "right";
    const earlier = earlierSide === "left" ? left : right;
    const later = earlierSide === "left" ? right : left;
    const bytes = later.total - earlier.total;
    return {
      bytes,
      percent: earlier.total === 0 ? 0 : (bytes / earlier.total) * 100,
      regression: bytes > 0,
      earlierSide,
      label: "Size change over time",
      columnLabel: "Size change over time",
      description: `Footprint size change from ${earlier.label} to ${later.label}; negative means the later release is smaller`,
    };
  }

  const bytes = right.total - left.total;
  return {
    bytes,
    percent: left.total === 0 ? 0 : (bytes / left.total) * 100,
    regression: bytes > 0,
    earlierSide: null,
    label: "B vs A",
    columnLabel: "B − A",
    description: "Total footprint difference from platform A to platform B",
  };
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

function updateComparisonChart(analysis) {
  const tools = analysis.contributions;
  elements.comparisonAxisNote.textContent = analysis.releaseMode
    ? "− smaller · + larger"
    : "− B smaller · + B larger";
  destroyChart("comparison");
  state.charts.comparison = new Chart(labeledChartCanvas(
    "#comparisonChart",
    `Size change by binary from ${analysis.from.label} to ${analysis.to.label}`,
  ), {
    type: "bar",
    data: {
      labels: tools.map((tool) => tool.name),
      datasets: [{
        label: analysis.releaseMode ? "Size change" : "B − A",
        data: tools.map((tool) => tool.contribution / MEBIBYTE),
        backgroundColor: tools.map((tool) => contributionColor(tool.contribution)),
        borderRadius: 2,
        maxBarThickness: 22,
      }],
    },
    options: {
      indexAxis: "y",
      responsive: true,
      maintainAspectRatio: false,
      interaction: { mode: "index", intersect: false },
      scales: {
        x: {
          ...divergingAxisOptions(analysis.releaseMode ? "Size change by binary (MiB)" : "B − A by binary (MiB)"),
        },
        y: { grid: { display: false } },
      },
      plugins: {
        legend: { display: false },
        tooltip: {
          callbacks: {
            label: (context) => `${context.dataset.label}: ${formatSignedMiB(context.parsed.x)}`,
            afterLabel: (context) => {
              const tool = tools[context.dataIndex];
              return [
                `${analysis.from.label}: ${formatBytes(tool.from)}`,
                `${analysis.to.label}: ${formatBytes(tool.to)}`,
                tool.from === 0 ? "New binary" : `Size change: ${formatPercent(sizeChangePercent(tool.from, tool.to))}`,
              ];
            },
          },
        },
      },
      layout: { padding: { left: 8, right: 8 } },
    },
  });
}

function contributionColor(value) {
  if (value === 0) {
    return COLORS.graphite;
  }
  return value < 0 ? COLORS.teal : COLORS.amber;
}

function divergingAxisOptions(title) {
  return {
    grid: {
      color: (context) => context.tick.value === 0 ? COLORS.inkSoft : COLORS.grid,
      lineWidth: (context) => context.tick.value === 0 ? 2 : 1,
    },
    border: { display: false },
    title: axisTitle(title),
    ticks: {
      maxTicksLimit: 7,
      callback: formatSignedAxisValue,
    },
  };
}

function formatSignedAxisValue(value) {
  if (value === 0) {
    return "0";
  }
  const digits = Math.abs(value) < 1 ? 2 : Math.abs(value) < 10 ? 1 : 0;
  return `${value > 0 ? "+" : ""}${Number(value.toFixed(digits))}`;
}

function formatSignedMiB(value) {
  const sign = value > 0 ? "+" : value < 0 ? "−" : "";
  return `${sign}${Math.abs(value).toFixed(1)} MiB`;
}

function updateComparisonTable(tools, earlierSide) {
  elements.compareTable.replaceChildren(...tools.map((tool) => {
    const row = document.createElement("tr");
    let delta = "—";
    let deltaClass = "";
    if (earlierSide) {
      const earlier = earlierSide === "left" ? tool.left : tool.right;
      const later = earlierSide === "left" ? tool.right : tool.left;
      if (earlier === null) {
        delta = "New";
        deltaClass = "delta-regression";
      } else if (later === null) {
        delta = "Removed";
        deltaClass = "delta-improvement";
      } else if (earlier > 0) {
        const percent = sizeChangePercent(earlier, later);
        delta = formatPercent(percent);
        deltaClass = percent > 0 ? "delta-regression" : "delta-improvement";
      }
    } else if (tool.left === null) {
      delta = "New";
      deltaClass = "delta-regression";
    } else if (tool.right === null) {
      delta = "Removed";
      deltaClass = "delta-improvement";
    } else if (tool.left > 0) {
      const percent = ((tool.right / tool.left) - 1) * 100;
      delta = formatPercent(percent);
      deltaClass = percent > 0 ? "delta-regression" : "delta-improvement";
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
    const change = previous ? sizeChangePercent(previous.executablePayload, row.executablePayload) : null;
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
      cell(change === null ? "—" : formatPercent(change), change === null ? "" : change > 0 ? "delta-regression" : "delta-improvement"),
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
  updateHeatmap();
  if (synchronize) {
    synchronizeComparisonFromTop();
  }
}

function commonChartOptions(tooltipCallbacks, onClick, stacked = false, axisKind = "bytes") {
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
        ticks: {
          maxRotation: 0,
          autoSkip: true,
          autoSkipPadding: 12,
          maxTicksLimit: 9,
          callback(value) {
            return minorVersionLabel(this.getLabelForValue(value));
          },
        },
      },
      y: {
        stacked,
        ...valueAxisOptions(axisKind),
      },
    },
    plugins: {
      legend: { position: "bottom", align: "start" },
      tooltip: {
        callbacks: typeof tooltipCallbacks === "function"
          ? { label: tooltipCallbacks }
          : tooltipCallbacks,
      },
    },
    layout: { padding: { right: 8 } },
  };
}

function valueAxisOptions(kind) {
  const base = {
    grid: { color: COLORS.grid },
    border: { display: false },
    ticks: { maxTicksLimit: 6 },
  };
  switch (kind) {
  case "index":
    return {
      ...base,
      title: axisTitle("Index (first = 100)"),
      ticks: { ...base.ticks, callback: (value) => Math.round(value) },
    };
  case "percent":
    return {
      ...base,
      beginAtZero: true,
      title: axisTitle("Size change (%)"),
      ticks: { ...base.ticks, callback: formatAxisPercent },
    };
  case "count":
    return {
      ...base,
      beginAtZero: true,
      ticks: { ...base.ticks, precision: 0 },
    };
  default:
    return byteAxisOptions();
  }
}

function axisTitle(text) {
  return {
    display: true,
    text,
    color: COLORS.inkSoft,
    font: { family: "IBM Plex Mono", size: 9, weight: "500" },
    padding: { top: 5 },
  };
}

function formatAxisPercent(value) {
  if (value === 0) {
    return "0";
  }
  return `${value > 0 ? "+" : ""}${Math.round(value)}%`;
}

function byteAxisOptions(horizontal = false) {
  return {
    beginAtZero: true,
    grid: { color: COLORS.grid },
    border: { display: false },
    title: {
      display: true,
      text: "MiB",
      color: COLORS.inkSoft,
      font: { family: "IBM Plex Mono", size: 9, weight: "500" },
      padding: { top: 5 },
    },
    ticks: {
      autoSkip: true,
      autoSkipPadding: 12,
      align: horizontal ? "inner" : "center",
      maxTicksLimit: 6,
      callback: (value) => horizontal && value === 0 ? "" : Math.round(value / MEBIBYTE),
    },
  };
}

function minorVersionLabel(version) {
  const parts = version.replace(/^go/, "").split(".");
  return parts.length >= 2 ? `${parts[0]}.${parts[1]}` : version;
}

function lineDataset(label, data, borderColor, backgroundColor, fill = true) {
  return {
    label,
    data,
    borderColor,
    backgroundColor,
    borderWidth: 2,
    pointBackgroundColor: (context) => state.releases[context.dataIndex]?.development ? COLORS.surface : borderColor,
    pointBorderColor: (context) => state.releases[context.dataIndex]?.development ? borderColor : COLORS.surface,
    pointBorderWidth: (context) => state.releases[context.dataIndex]?.development ? 3 : 1,
    pointRadius: (context) => state.releases[context.dataIndex]?.development ? 6 : 3,
    pointStyle: (context) => state.releases[context.dataIndex]?.development ? "rectRot" : "circle",
    pointHoverRadius: 5,
    tension: 0.22,
    spanGaps: false,
    fill,
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

function labeledChartCanvas(selector, label) {
  const canvas = document.querySelector(selector);
  canvas.setAttribute("aria-label", label);
  return canvas;
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
  return `${platform.os}/${canonicalArch(platform.arch)}`;
}

function platformLabel(key) {
  const [os, arch] = key.split("/");
  const canonical = canonicalArch(arch);
  return `${OS_NAMES[os] || os} / ${ARCH_NAMES[canonical] || canonical}`;
}

function canonicalArch(arch) {
  return arch === "armv6" || arch === "armv6l" ? "arm" : arch;
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
  const stableCount = state.releases.filter((release) => !release.development).length;
  const measurementCount = sum(state.releases.flatMap((release) => release.platforms).map((platform) => platform.tools.length));
  elements.datasetRange.textContent = state.tipRelease
    ? `${first} → ${latest} · ${stableCount} stable + tip · ${new Intl.NumberFormat("en").format(measurementCount)} binaries`
    : `${first} → ${latest} · ${stableCount} releases · ${new Intl.NumberFormat("en").format(measurementCount)} binaries`;
  const generated = new Date(state.report.generated_at);
  elements.generatedAt.dateTime = state.report.generated_at;
  elements.generatedAt.textContent = `Generated ${new Intl.DateTimeFormat("en", { dateStyle: "medium", timeZone: "UTC" }).format(generated)} UTC`;
}

function updateTipNotice() {
  if (!state.tipRelease) {
    elements.tipNotice.hidden = true;
    return;
  }
  const release = state.tipRelease;
  const revision = release.revision || "unknown";
  elements.tipVersion.textContent = release.version;
  elements.tipRevision.textContent = revision.slice(0, 12);
  elements.tipRevision.title = revision;
  elements.tipPlatforms.textContent = pluralize(release.platforms.length, "platform");
  if (release.commit_time) {
    const commitTime = new Date(release.commit_time);
    elements.tipCommitTime.dateTime = release.commit_time;
    elements.tipCommitTime.textContent = `${new Intl.DateTimeFormat("en", { dateStyle: "medium", timeStyle: "short", timeZone: "UTC" }).format(commitTime)} UTC`;
  }
  const repository = (release.source || "https://github.com/golang/go.git").replace(/\.git$/, "");
  elements.tipSource.href = revision !== "unknown" ? `${repository}/commit/${revision}` : repository;
  elements.tipNotice.hidden = false;
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

function pluralize(count, noun, plural = `${noun}s`) {
  return `${count} ${count === 1 ? noun : plural}`;
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
  return compareAlphaNumeric(platformLabel(left), platformLabel(right));
}

function compareAlphaNumeric(left, right) {
  return left.localeCompare(right, "en", {
    numeric: true,
    sensitivity: "base",
  });
}

function updateHeatmap() {
  elements.heatmapNote.textContent = state.heatmapMode === "absolute"
    ? "Color intensity uses the 5th–95th percentile; hatched cells have no measurement."
    : "Teal is smaller; amber is larger than prior. Intensity uses the 5th–95th percentile; hatching means no comparison.";
  elements.heatmap.setAttribute(
    "aria-label",
    state.heatmapMode === "absolute"
      ? "Executable footprint by platform and release"
      : "Size change from prior release by platform",
  );
  const platformKeys = [...new Set(state.releases.flatMap((release) => release.platforms.map(platformKey)))].sort(comparePlatformKeys);
  const matrix = platformKeys.map((key) => ({
    key,
    values: state.releases.map((release) => {
      const platform = preferredPlatform(release, key);
      return platform ? sum(platform.tools.map((tool) => tool.size)) : null;
    }),
  }));
  const displayValues = [];
  for (const row of matrix) {
    for (let index = 0; index < row.values.length; index += 1) {
      const value = heatmapValue(row.values, index);
      if (value !== null) {
        displayValues.push(value);
      }
    }
  }
  const low = quantile(displayValues, 0.05);
  const high = quantile(displayValues, 0.95);

  const grid = document.createElement("div");
  grid.className = "heatmap__grid";
  grid.style.setProperty("--release-columns", state.releases.length);
  grid.append(heatmapCorner());
  for (const release of state.releases) {
    const header = document.createElement("span");
    header.className = "heatmap__column-label";
    header.textContent = minorVersionLabel(release.version);
    header.title = release.version;
    header.setAttribute("role", "columnheader");
    grid.append(header);
  }

  for (const row of matrix) {
    const rowLabel = document.createElement("span");
    rowLabel.className = "heatmap__row-label";
    rowLabel.textContent = platformLabel(row.key);
    rowLabel.setAttribute("role", "rowheader");
    grid.append(rowLabel);
    for (let index = 0; index < state.releases.length; index += 1) {
      const release = state.releases[index];
      const raw = row.values[index];
      const display = heatmapValue(row.values, index);
      if (display === null) {
        const missing = document.createElement("span");
        missing.className = "heatmap__cell heatmap__cell--missing";
        missing.setAttribute("role", "gridcell");
        missing.setAttribute("aria-label", `${platformLabel(row.key)}, ${release.version}: unavailable`);
        missing.title = `${platformLabel(row.key)} · ${release.version}\nUnavailable`;
        grid.append(missing);
        continue;
      }

      const cell = document.createElement("button");
      cell.className = "heatmap__cell";
      if (row.key === state.platform && release.version === state.release) {
        cell.classList.add("heatmap__cell--selected");
      }
      cell.type = "button";
      cell.style.backgroundColor = heatmapColor(display, low, high);
      const valueLabel = state.heatmapMode === "absolute" ? formatBytes(raw) : `Size change: ${formatPercent(display)}`;
      cell.setAttribute("role", "gridcell");
      cell.setAttribute("aria-label", `${platformLabel(row.key)}, ${release.version}: ${valueLabel}`);
      cell.title = `${platformLabel(row.key)} · ${release.version}\n${valueLabel}`;
      cell.addEventListener("click", () => selectHeatmapCell(row.key, release.version));
      grid.append(cell);
    }
  }

  elements.heatmap.replaceChildren(grid);
  updateHeatmapLegend(low, high);
}

function heatmapCorner() {
  const corner = document.createElement("span");
  corner.className = "heatmap__corner";
  corner.textContent = "Platform";
  return corner;
}

function heatmapValue(values, index) {
  const value = values[index];
  if (value === null) {
    return null;
  }
  if (state.heatmapMode === "absolute") {
    return value;
  }
  if (index === 0 || values[index - 1] === null) {
    return null;
  }
  return sizeChangePercent(values[index - 1], value);
}

function heatmapColor(value, low, high) {
  if (state.heatmapMode === "delta") {
    const extent = Math.max(Math.abs(low), Math.abs(high), 0.1);
    const intensity = Math.min(1, Math.abs(value) / extent);
    const rgb = value < 0 ? [43, 127, 116] : value > 0 ? [212, 130, 24] : [126, 147, 154];
    return `rgba(${rgb.join(",")},${(0.16 + intensity * 0.78).toFixed(2)})`;
  }
  const position = high === low ? 0.5 : Math.max(0, Math.min(1, (value - low) / (high - low)));
  return `rgba(0,173,216,${(0.14 + position * 0.82).toFixed(2)})`;
}

function quantile(values, probability) {
  if (values.length === 0) {
    return 0;
  }
  const sorted = [...values].sort((left, right) => left - right);
  const position = (sorted.length - 1) * probability;
  const lower = Math.floor(position);
  const fraction = position - lower;
  return sorted[lower + 1] === undefined
    ? sorted[lower]
    : sorted[lower] + fraction * (sorted[lower + 1] - sorted[lower]);
}

function updateHeatmapLegend(low, high) {
  const lowLabel = state.heatmapMode === "absolute" ? formatBytes(low) : formatPercent(low);
  const highLabel = state.heatmapMode === "absolute" ? formatBytes(high) : formatPercent(high);
  elements.heatmapLegend.innerHTML = "";
  const start = document.createElement("span");
  start.textContent = lowLabel;
  const scale = document.createElement("span");
  scale.className = `heatmap-legend__scale heatmap-legend__scale--${state.heatmapMode}`;
  const end = document.createElement("span");
  end.textContent = highLabel;
  const missing = document.createElement("span");
  missing.className = "heatmap-legend__missing";
  missing.textContent = "Unavailable";
  elements.heatmapLegend.append(start, scale, end, missing);
}

function selectHeatmapCell(platform, release) {
  state.platform = platform;
  state.release = release;
  elements.platformSelect.value = platform;
  updatePlatform();
}