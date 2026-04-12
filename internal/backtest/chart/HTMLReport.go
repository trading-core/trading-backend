package chart

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
	"time"
)

// RenderHTMLReportInput extends the combined chart input with backtest summary stats.
type RenderHTMLReportInput struct {
	Symbol       string
	TotalReturn  float64
	StartingCash float64
	EndingCash   float64
	EndingValue  float64
	TradeCount   int
	WinRate      float64
	SharpeRatio  float64
	Prices       []PricePoint
	Decisions    []DecisionMarker
	BollUpper    []IndicatorPoint
	BollMiddle   []IndicatorPoint
	BollLower    []IndicatorPoint
	SMA          []IndicatorPoint
	RSI          []IndicatorPoint
	MACD         []IndicatorPoint
	MACDSignal   []IndicatorPoint
	ATR          []IndicatorPoint
	SMAPeriod    int
	RSIPeriod    int
	MACDFast     int
	MACDSlow     int
	MACDSignalN  int
	ATRPeriod    int
	Timezone     *time.Location
	Timeframe    string
}

type htmlPoint struct {
	T string  `json:"t"`
	V float64 `json:"v"`
}

type htmlDecision struct {
	T      string  `json:"t"`
	Price  float64 `json:"price"`
	Qty    float64 `json:"qty"`
	IsBuy  bool    `json:"is_buy"`
	Reason string  `json:"reason"`
}

type htmlReportData struct {
	Symbol       string         `json:"symbol"`
	TotalReturn  float64        `json:"total_return"`
	StartingCash float64        `json:"starting_cash"`
	EndingCash   float64        `json:"ending_cash"`
	EndingValue  float64        `json:"ending_value"`
	TradeCount   int            `json:"trade_count"`
	WinRate      float64        `json:"win_rate"`
	SharpeRatio  float64        `json:"sharpe_ratio"`
	Prices       []htmlPoint    `json:"prices"`
	BollUpper    []htmlPoint    `json:"boll_upper"`
	BollMiddle   []htmlPoint    `json:"boll_middle"`
	BollLower    []htmlPoint    `json:"boll_lower"`
	SMA          []htmlPoint    `json:"sma"`
	RSI          []htmlPoint    `json:"rsi"`
	MACD         []htmlPoint    `json:"macd"`
	MACDSignal   []htmlPoint    `json:"macd_signal"`
	ATR          []htmlPoint    `json:"atr"`
	Decisions    []htmlDecision `json:"decisions"`
	SMAPeriod    int            `json:"sma_period"`
	RSIPeriod    int            `json:"rsi_period"`
	MACDFast     int            `json:"macd_fast"`
	MACDSlow     int            `json:"macd_slow"`
	MACDSignalN  int            `json:"macd_signal_n"`
	ATRPeriod    int            `json:"atr_period"`
}

func toHTMLPoints(pts []IndicatorPoint, tz *time.Location) []htmlPoint {
	out := make([]htmlPoint, len(pts))
	for i, p := range pts {
		out[i] = htmlPoint{T: p.At.In(tz).Format("2006-01-02"), V: p.Value}
	}
	return out
}

// RenderHTMLReport generates a self-contained interactive HTML report at outputPath.
// The report embeds all data as JSON and uses Plotly.js (loaded from CDN) to render
// three stacked panels — price with overlays, RSI, and MACD — with hover tooltips on
// each buy/sell decision marker showing the reason, price, and quantity.
func RenderHTMLReport(input RenderHTMLReportInput, outputPath string) error {
	tz := input.Timezone
	if tz == nil {
		tz = time.UTC
	}

	decisions := make([]htmlDecision, len(input.Decisions))
	for i, d := range input.Decisions {
		decisions[i] = htmlDecision{
			T:      d.At.In(tz).Format("2006-01-02"),
			Price:  d.Price,
			Qty:    d.Quantity,
			IsBuy:  d.IsBuy,
			Reason: d.Reason,
		}
	}

	prices := make([]htmlPoint, len(input.Prices))
	for i, p := range input.Prices {
		prices[i] = htmlPoint{T: p.At.In(tz).Format("2006-01-02"), V: p.Close}
	}

	data := htmlReportData{
		Symbol:       input.Symbol,
		TotalReturn:  input.TotalReturn,
		StartingCash: input.StartingCash,
		EndingCash:   input.EndingCash,
		EndingValue:  input.EndingValue,
		TradeCount:   input.TradeCount,
		WinRate:      input.WinRate,
		SharpeRatio:  input.SharpeRatio,
		Prices:       prices,
		BollUpper:    toHTMLPoints(input.BollUpper, tz),
		BollMiddle:   toHTMLPoints(input.BollMiddle, tz),
		BollLower:    toHTMLPoints(input.BollLower, tz),
		SMA:          toHTMLPoints(input.SMA, tz),
		RSI:          toHTMLPoints(input.RSI, tz),
		MACD:         toHTMLPoints(input.MACD, tz),
		MACDSignal:   toHTMLPoints(input.MACDSignal, tz),
		ATR:          toHTMLPoints(input.ATR, tz),
		Decisions:    decisions,
		SMAPeriod:    input.SMAPeriod,
		RSIPeriod:    input.RSIPeriod,
		MACDFast:     input.MACDFast,
		MACDSlow:     input.MACDSlow,
		MACDSignalN:  input.MACDSignalN,
		ATRPeriod:    input.ATRPeriod,
	}

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal report data: %w", err)
	}

	tmpl, err := template.New("report").Parse(htmlReportTemplate)
	if err != nil {
		return fmt.Errorf("parse html template: %w", err)
	}

	outputDir := filepath.Dir(outputPath)
	if outputDir != "." && outputDir != "" {
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			return err
		}
	}
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, map[string]any{
		"DataJSON": string(dataJSON),
	})
}

const htmlReportTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Backtest Report</title>
<script src="https://cdn.plot.ly/plotly-2.26.0.min.js"></script>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; background: #f5f5f5; color: #222; }
  .header { background: #1a1a2e; color: #fff; padding: 18px 28px; display: flex; align-items: baseline; gap: 16px; }
  .header h1 { font-size: 1.5rem; font-weight: 700; letter-spacing: 0.04em; }
  .header .sub { font-size: 0.9rem; color: #aab; }
  .stats { display: flex; background: #fff; border-bottom: 1px solid #e0e0e0; }
  .stat { padding: 12px 24px; border-right: 1px solid #e8e8e8; }
  .stat .label { font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.06em; color: #888; margin-bottom: 3px; }
  .stat .value { font-size: 1.1rem; font-weight: 600; }
  .value.pos { color: #1a7a3a; }
  .value.neg { color: #c0392b; }

  /* Each panel is an independent div; the handle between them resizes only the panel above it */
  #chart-wrap { background: #fff; user-select: none; }
  .panel { width: 100%; overflow: hidden; }
  #panel-price { height: 480px; }
  #panel-rsi   { height: 480px; }
  #panel-macd  { height: 480px; }
  #panel-atr   { height: 200px; }

  .drag-handle {
    width: 100%; height: 6px; cursor: ns-resize;
    background: #ebebeb;
    display: flex; align-items: center; justify-content: center;
    transition: background 0.15s;
  }
  .drag-handle::after {
    content: ''; display: block;
    width: 36px; height: 3px; border-radius: 2px;
    background: #ccc; transition: background 0.15s;
  }
  .drag-handle:hover, .drag-handle.dragging { background: #dde7ff; }
  .drag-handle:hover::after, .drag-handle.dragging::after { background: #5b8dee; }

  .decisions-section { padding: 24px 28px; }
  .decisions-section h2 { font-size: 1rem; font-weight: 600; margin-bottom: 12px; color: #444; }
  table { width: 100%; border-collapse: collapse; font-size: 0.85rem; background: #fff;
          border-radius: 6px; overflow: hidden; box-shadow: 0 1px 4px rgba(0,0,0,0.08); }
  th { background: #f0f0f5; padding: 9px 14px; text-align: left; font-weight: 600; color: #555; border-bottom: 2px solid #ddd; }
  td { padding: 8px 14px; border-bottom: 1px solid #f0f0f0; vertical-align: middle; }
  tr:last-child td { border-bottom: none; }
  tr:hover td { background: #f9f9ff; }
  .badge { display: inline-block; padding: 2px 10px; border-radius: 12px; font-size: 0.75rem; font-weight: 700; letter-spacing: 0.04em; }
  .badge.buy  { background: #e6f7ee; color: #1a7a3a; }
  .badge.sell { background: #fdecea; color: #c0392b; }
  .reason { color: #555; font-style: italic; }
</style>
</head>
<body>

<div class="header">
  <h1 id="hdr-symbol"></h1>
  <span class="sub" id="hdr-sub"></span>
</div>
<div class="stats">
  <div class="stat"><div class="label">Total Return</div><div class="value" id="stat-return"></div></div>
  <div class="stat"><div class="label">Starting Cash</div><div class="value" id="stat-start"></div></div>
  <div class="stat"><div class="label">Ending Value</div><div class="value" id="stat-end"></div></div>
  <div class="stat"><div class="label">Trades</div><div class="value" id="stat-trades"></div></div>
  <div class="stat"><div class="label">Win Rate</div><div class="value" id="stat-winrate"></div></div>
  <div class="stat"><div class="label">Sharpe Ratio</div><div class="value" id="stat-sharpe"></div></div>
</div>

<div id="chart-wrap">
  <div id="panel-price" class="panel"></div>
  <div class="drag-handle" data-resizes="panel-price"></div>
  <div id="panel-rsi"   class="panel"></div>
  <div class="drag-handle" data-resizes="panel-rsi"></div>
  <div id="panel-macd"  class="panel"></div>
  <div class="drag-handle" data-resizes="panel-macd"></div>
  <div id="panel-atr"   class="panel"></div>
  <div class="drag-handle" data-resizes="panel-atr"></div>
</div>

<div class="decisions-section">
  <h2>Trade Log</h2>
  <table>
    <thead><tr><th>Date</th><th>Action</th><th>Price</th><th>Quantity</th><th>Reason</th></tr></thead>
    <tbody id="decisions-body"></tbody>
  </table>
</div>

<script>
const report = {{.DataJSON}};

// ── Header & stats ───────────────────────────────────────────────────────────
document.getElementById('hdr-symbol').textContent = report.symbol;
document.getElementById('hdr-sub').textContent =
  'RSI(' + report.rsi_period + ')  MACD(' + report.macd_fast + ',' +
  report.macd_slow + ',' + report.macd_signal_n + ')  SMA(' + report.sma_period + ')' +
  '  ATR(' + report.atr_period + ')';

function fmt$(v)  { return '$' + v.toLocaleString('en-US', {minimumFractionDigits:2, maximumFractionDigits:4}); }
function fmtPct(v){ return (v >= 0 ? '+' : '') + (v * 100).toFixed(2) + '%'; }
function fmtQty(v){ return v.toLocaleString('en-US', {maximumFractionDigits:0}); }

const retEl = document.getElementById('stat-return');
retEl.textContent = fmtPct(report.total_return);
retEl.className = 'value ' + (report.total_return >= 0 ? 'pos' : 'neg');
document.getElementById('stat-start').textContent  = fmt$(report.starting_cash);
document.getElementById('stat-end').textContent    = fmt$(report.ending_value);
document.getElementById('stat-trades').textContent = report.trade_count;
document.getElementById('stat-winrate').textContent= (report.win_rate * 100).toFixed(1) + '%';
document.getElementById('stat-sharpe').textContent = report.sharpe_ratio.toFixed(2);

// ── Trade log ────────────────────────────────────────────────────────────────
const tbody = document.getElementById('decisions-body');
report.decisions.forEach(d => {
  const side = d.is_buy ? 'buy' : 'sell';
  const tr = document.createElement('tr');
  tr.innerHTML =
    '<td>' + d.t + '</td>' +
    '<td><span class="badge ' + side + '">' + side.toUpperCase() + '</span></td>' +
    '<td>' + fmt$(d.price) + '</td>' +
    '<td>' + fmtQty(d.qty) + '</td>' +
    '<td class="reason">' + d.reason + '</td>';
  tbody.appendChild(tr);
});

// ── Data alignment ────────────────────────────────────────────────────────────
// All traces share priceDates as the x-axis (category type).
// This mirrors the PNG's index-based approach: no weekend/holiday gaps,
// MACD histogram bars are gapless, and bargap:0 works correctly.
const priceDates = report.prices.map(p => p.t);
const priceIdx = Object.create(null);
priceDates.forEach((d, i) => { priceIdx[d] = i; });
function align(pts) {
  const ys = new Array(priceDates.length).fill(null);
  pts.forEach(p => { const i = priceIdx[p.t]; if (i !== undefined) ys[i] = p.v; });
  return ys;
}

// ── Decision markers ─────────────────────────────────────────────────────────
const buys  = report.decisions.filter(d =>  d.is_buy);
const sells = report.decisions.filter(d => !d.is_buy);
function hoverText(d) {
  return (d.is_buy ? 'BUY' : 'SELL') + ' @ ' + fmt$(d.price) +
    ' × ' + fmtQty(d.qty) + '<br>' + d.reason + '<br>' + d.t;
}

// ── MACD histogram (bar trace, TradingView 4-colour style) ───────────────────
// Bar traces on category axes respect bargap:0 perfectly — no gaps possible.
const macdValMap = Object.create(null), sigValMap = Object.create(null);
report.macd.forEach(p => { macdValMap[p.t] = p.v; });
report.macd_signal.forEach(p => { sigValMap[p.t] = p.v; });
const histVals   = new Array(priceDates.length).fill(null);
const histColors = new Array(priceDates.length).fill('rgba(0,0,0,0)');
let prevH = null;
priceDates.forEach((d, i) => {
  if (macdValMap[d] === undefined || sigValMap[d] === undefined) return;
  const h = macdValMap[d] - sigValMap[d];
  histVals[i] = h;
  if (h >= 0) {
    histColors[i] = (prevH !== null && h >= prevH) ? 'rgba(34,180,34,0.75)' : 'rgba(144,220,144,0.75)';
  } else {
    histColors[i] = (prevH !== null && h <= prevH) ? 'rgba(220,40,40,0.75)' : 'rgba(240,160,160,0.75)';
  }
  prevH = h;
});

// ── Shared layout helpers ────────────────────────────────────────────────────
const ML = 70, MR = 30;
const xAxisBase = {
  type: 'category',   // index-based: no weekend/holiday gaps, matches PNG behaviour
  showgrid: true, gridcolor: '#e8e8e8',
  zeroline: false, rangeslider: {visible: false},
  showspikes: true, spikemode: 'across', spikecolor: '#999',
  spikethickness: 1, spikedash: 'dot',
  nticks: 12,
};
const config = {
  responsive: true, displayModeBar: true,
  modeBarButtonsToRemove: ['select2d','lasso2d','autoScale2d'],
};
const configNoBar = { responsive: true, displayModeBar: false };
const hoverLabel = { bgcolor:'#1a1a2e', bordercolor:'#555', font:{color:'#fff', size:13} };

function panelH(id) { return document.getElementById(id).offsetHeight; }

// ── Plot: Price ───────────────────────────────────────────────────────────────
const plotPrice = Plotly.newPlot('panel-price', [
  { x:priceDates, y:report.prices.map(p=>p.v), mode:'lines', type:'scatter', name:'Price',
    line:{color:'#2378e6',width:1.5} },
  { x:priceDates, y:align(report.boll_upper),  mode:'lines', type:'scatter', name:'BB Upper',
    line:{color:'#b85a18',width:1,dash:'dot'} },
  { x:priceDates, y:align(report.boll_middle), mode:'lines', type:'scatter', name:'BB Mid',
    line:{color:'#777',width:1,dash:'dash'} },
  { x:priceDates, y:align(report.boll_lower),  mode:'lines', type:'scatter', name:'BB Lower',
    line:{color:'#189068',width:1,dash:'dot'} },
  { x:priceDates, y:align(report.sma), mode:'lines', type:'scatter',
    name:'SMA('+report.sma_period+')', line:{color:'#8230c8',width:1.2} },
  { x:buys.map(d=>d.t),  y:buys.map(d=>d.price),  mode:'markers', type:'scatter', name:'Buy',
    marker:{symbol:'triangle-up', color:'#1a7a3a', size:12, line:{color:'#0d5c28',width:1.5}},
    text:buys.map(hoverText), hoverinfo:'text' },
  { x:sells.map(d=>d.t), y:sells.map(d=>d.price), mode:'markers', type:'scatter', name:'Sell',
    marker:{symbol:'circle-open', color:'#c0392b', size:11, line:{color:'#c0392b',width:2.5}},
    text:sells.map(hoverText), hoverinfo:'text' },
], {
  paper_bgcolor:'#fff', plot_bgcolor:'#fff',
  margin:{l:ML, r:MR, t:20, b:0},
  height: panelH('panel-price'),
  xaxis: Object.assign({}, xAxisBase, {showticklabels:false}),
  yaxis: {showgrid:true, gridcolor:'#e8e8e8', zeroline:false, tickprefix:'$', title:{text:'Price',standoff:8}},
  legend:{x:0, y:1.01, xanchor:'left', yanchor:'bottom', orientation:'h',
          bgcolor:'rgba(255,255,255,0.8)', bordercolor:'#ddd', borderwidth:1},
  hoverlabel: hoverLabel, hovermode:'closest',
  dragmode: 'pan',
}, config);

// ── Plot: RSI ───────────────────────────────────────────────────────────────
const rsiAligned = align(report.rsi);
// Clamp RSI to above 70 (so fill between the 70 line and RSI only where overbought)
const rsiOB = rsiAligned.map(v => v !== null ? Math.max(v, 70) : null);
// Clamp RSI to below 30 (so fill between RSI and the 30 line only where oversold)
const rsiOS = rsiAligned.map(v => v !== null ? Math.min(v, 30) : null);
const ref70 = new Array(priceDates.length).fill(70);
const ref30 = new Array(priceDates.length).fill(30);

const plotRSI = Plotly.newPlot('panel-rsi', [
  // Overbought fill: tonexty fills from ref70 up to rsiOB
  { x:priceDates, y:ref70, mode:'lines', type:'scatter', showlegend:false,
    line:{color:'transparent',width:0}, hoverinfo:'skip' },
  { x:priceDates, y:rsiOB, mode:'lines', type:'scatter', showlegend:false,
    fill:'tonexty', fillcolor:'rgba(34,160,60,0.22)',
    line:{color:'transparent',width:0}, hoverinfo:'skip' },
  // Oversold fill: tonexty fills from rsiOS up to ref30
  { x:priceDates, y:rsiOS, mode:'lines', type:'scatter', showlegend:false,
    line:{color:'transparent',width:0}, hoverinfo:'skip' },
  { x:priceDates, y:ref30, mode:'lines', type:'scatter', showlegend:false,
    fill:'tonexty', fillcolor:'rgba(210,50,50,0.22)',
    line:{color:'transparent',width:0}, hoverinfo:'skip' },
  // RSI line on top
  { x:priceDates, y:rsiAligned, mode:'lines', type:'scatter', name:'RSI',
    line:{color:'#a0461e',width:1.5}, showlegend:false },
], {
  paper_bgcolor:'#fff', plot_bgcolor:'#fff',
  margin:{l:ML, r:MR, t:6, b:0},
  height: panelH('panel-rsi'),
  bargap: 0,
  xaxis: Object.assign({}, xAxisBase, {showticklabels:false}),
  yaxis: {showgrid:true, gridcolor:'#e8e8e8', zeroline:false, range:[0,100],
          title:{text:'RSI',standoff:8},
          tickvals:[30,50,70]},
  shapes: [
    {type:'line', xref:'paper', yref:'y', x0:0,x1:1, y0:30,y1:30, line:{color:'rgba(60,140,60,0.4)',width:1,dash:'dot'}},
    {type:'line', xref:'paper', yref:'y', x0:0,x1:1, y0:70,y1:70, line:{color:'rgba(200,50,50,0.4)',width:1,dash:'dot'}},
  ],
  hoverlabel: hoverLabel, hovermode:'x',
  dragmode: 'pan',
}, configNoBar);

// ── Plot: MACD ───────────────────────────────────────────────────────────────
const plotMACD = Plotly.newPlot('panel-macd', [
  { x:priceDates, y:histVals, type:'bar', name:'Histogram',
    marker:{color:histColors, line:{width:0}}, showlegend:false },
  { x:priceDates, y:align(report.macd),        mode:'lines', type:'scatter', name:'MACD',
    line:{color:'#2378e6',width:1.5}, showlegend:false },
  { x:priceDates, y:align(report.macd_signal), mode:'lines', type:'scatter', name:'Signal',
    line:{color:'#dc2828',width:1.5}, showlegend:false },
], {
  paper_bgcolor:'#fff', plot_bgcolor:'#fff',
  margin:{l:ML, r:MR, t:6, b:0},
  height: panelH('panel-macd'),
  bargap: 0,
  xaxis: Object.assign({}, xAxisBase, {showticklabels:false}),
  yaxis: {showgrid:true, gridcolor:'#e8e8e8', zeroline:true, zerolinecolor:'#bbb',
          title:{text:'MACD',standoff:8}},
  hoverlabel: hoverLabel, hovermode:'x',
  dragmode: 'pan',
}, configNoBar);

// ── Plot: ATR ────────────────────────────────────────────────────────────────
const plotATR = Plotly.newPlot('panel-atr', [
  { x:priceDates, y:align(report.atr), mode:'lines', type:'scatter',
    name:'ATR('+report.atr_period+')',
    line:{color:'#14a0a0',width:1.5}, showlegend:false },
], {
  paper_bgcolor:'#fff', plot_bgcolor:'#fff',
  margin:{l:ML, r:MR, t:6, b:40},
  height: panelH('panel-atr'),
  xaxis: Object.assign({}, xAxisBase, {showticklabels:true}),
  yaxis: {showgrid:true, gridcolor:'#e8e8e8', zeroline:false,
          title:{text:'ATR('+report.atr_period+')',standoff:8}},
  hoverlabel: hoverLabel, hovermode:'x',
  dragmode: 'pan',
}, configNoBar);

// ── Initial x-axis alignment ──────────────────────────────────────────────────
// Each chart auto-ranges independently on first render. Sync all panels to
// the price panel's computed range once all plots have finished.
Promise.all([plotPrice, plotRSI, plotMACD, plotATR]).then(() => {
  const range = document.getElementById('panel-price')._fullLayout.xaxis.range;
  if (range) {
    Plotly.relayout('panel-rsi',  {'xaxis.range': range});
    Plotly.relayout('panel-macd', {'xaxis.range': range});
    Plotly.relayout('panel-atr',  {'xaxis.range': range});
  }
});

// ── X-axis sync across all four panels ───────────────────────────────────────
// Sync is batched to one animation frame so rapid pan/zoom events don't stack
// up relayout calls and freeze the browser.
const PANEL_IDS = ['panel-price', 'panel-rsi', 'panel-macd', 'panel-atr'];
let syncing = false;
let syncRaf  = null;   // pending rAF id
let syncSrc  = null;   // panel that triggered the latest update
let syncRange = null;  // latest range, or null for autorange

PANEL_IDS.forEach(srcId => {
  document.getElementById(srcId).on('plotly_relayout', ev => {
    if (syncing) return;

    // Sync dragmode from the price panel (which has the modebar) to the others.
    if (srcId === 'panel-price' && ev['dragmode'] !== undefined) {
      syncing = true;
      PANEL_IDS.filter(id => id !== 'panel-price').forEach(id =>
        Plotly.relayout(id, { dragmode: ev['dragmode'] })
      );
      syncing = false;
    }

    let range = null;
    if (ev['xaxis.range[0]'] !== undefined) {
      range = [ev['xaxis.range[0]'], ev['xaxis.range[1]']];
    } else if (Array.isArray(ev['xaxis.range'])) {
      range = ev['xaxis.range'];
    }
    const autorange = ev['xaxis.autorange'] === true;
    if (!range && !autorange) return;

    // Record latest intent; coalesce multiple events into one rAF.
    syncSrc   = srcId;
    syncRange = range; // null means autorange

    if (syncRaf) return; // already scheduled for this frame
    syncRaf = requestAnimationFrame(() => {
      syncRaf = null;
      syncing = true;
      const update = syncRange ? {'xaxis.range': syncRange} : {'xaxis.autorange': true};
      PANEL_IDS.filter(id => id !== syncSrc).forEach(id => Plotly.relayout(id, update));
      syncing = false;
    });
  });
});

// ── Drag-to-resize handles ────────────────────────────────────────────────────
// Each handle carries data-resizes="<panel-id>". Dragging adjusts only that panel's
// height; panels below shift naturally in the document flow — nothing is taken from them.
const MIN_PANEL_PX = 80;
let drag = null;

document.querySelectorAll('.drag-handle').forEach(handle => {
  handle.addEventListener('mousedown', e => {
    e.preventDefault();
    const targetId = handle.dataset.resizes;
    const targetEl = document.getElementById(targetId);
    drag = { handle, targetId, targetEl, startY: e.clientY, startH: targetEl.offsetHeight };
    handle.classList.add('dragging');
    document.body.style.cursor = 'ns-resize';
  });
});

document.addEventListener('mousemove', e => {
  if (!drag) return;
  const newH = Math.max(MIN_PANEL_PX, drag.startH + (e.clientY - drag.startY));
  drag.targetEl.style.height = newH + 'px';
  Plotly.relayout(drag.targetId, { height: newH });
});

document.addEventListener('mouseup', () => {
  if (!drag) return;
  drag.handle.classList.remove('dragging');
  document.body.style.cursor = '';
  drag = null;
});
</script>
</body>
</html>
`
