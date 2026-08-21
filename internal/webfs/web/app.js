'use strict';
// Reliability analysis workbench. Plain fetch + DOM, no framework.
const API = '';
let currentAID = '';

async function j(method, path, body) {
  const opt = { method, headers: {} };
  if (body !== undefined && body !== null) {
    opt.headers['Content-Type'] = 'application/json';
    opt.body = JSON.stringify(body);
  }
  const r = await fetch(API + path, opt);
  const txt = await r.text();
  let data = null;
  if (txt) { try { data = JSON.parse(txt); } catch (e) { data = txt; } }
  if (!r.ok) throw new Error((data && data.error) || ('HTTP ' + r.status));
  return data;
}

function short(id) { return id ? id.slice(0, 8) : ''; }
function esc(s) { return String(s || '').replace(/[&<>]/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;' }[c])); }
function setAID(id) {
  currentAID = id;
  for (const el of ['f-aid', 'm-aid', 'r-aid', 'd-aid', 's-aid']) {
    const e = document.getElementById(el); if (e) e.value = id;
  }
}

// --- tabs ---
document.querySelectorAll('.tab').forEach(t => {
  t.addEventListener('click', () => {
    document.querySelectorAll('.tab').forEach(x => x.classList.remove('active'));
    document.querySelectorAll('.panel').forEach(x => x.classList.remove('active'));
    t.classList.add('active');
    document.getElementById('tab-' + t.dataset.tab).classList.add('active');
  });
});

// --- analyses ---
async function loadAnalyses() {
  const list = await j('GET', '/analyses');
  const tb = document.querySelector('#a-table tbody');
  tb.innerHTML = '';
  for (const a of list || []) {
    const tr = document.createElement('tr');
    tr.className = 'clickable';
    tr.innerHTML = `<td>${short(a.id)}</td><td>${esc(a.name)}</td><td>${a.state}</td><td>v${a.version}</td><td>${esc(a.top_event)}</td>`;
    tr.addEventListener('click', () => { setAID(a.id); loadAllFor(a.id); });
    tb.appendChild(tr);
  }
}
document.getElementById('a-refresh').addEventListener('click', loadAnalyses);
document.getElementById('a-create').addEventListener('click', async () => {
  const name = document.getElementById('a-name').value.trim();
  const top = document.getElementById('a-top').value.trim();
  if (!name) return alert('请输入名称');
  const a = await j('POST', '/analyses', { name, top_event: top });
  setAID(a.id); await loadAnalyses(); await loadAllFor(a.id);
});

async function loadAllFor(id) {
  await loadFTA(id); await loadFMEA(id); await loadRBD(id); await loadFDist(id);
}

// --- FTA in-memory editor ---
let gates = [], events = [], inputs = {};
function resetFTA() { gates = []; events = []; inputs = {}; renderFTA(); }
async function loadFTA(id) {
  try {
    const t = await j('GET', `/analyses/${id}/fta`);
    gates = t.gates || []; events = t.events || [];
    inputs = {};
    for (const g of gates) { inputs[g.id] = g.inputs || []; }
    renderFTA();
  } catch (e) { resetFTA(); }
}
function renderFTA() {
  let s = '门:\n';
  for (const g of gates) {
    s += `  ${g.id} ${g.type}${g.k ? (' k=' + g.k) : ''}${g.is_top ? ' [TOP]' : ''} <- ${(inputs[g.id] || []).map(i => i.gate_id || i.event_id).join(', ')}\n`;
  }
  s += '基本事件:\n';
  for (const e of events) {
    s += `  ${e.id} ${e.label || ''} q=${((e.prob_micro || 0) / 1e6).toFixed(6)}${e.ccf_group ? (' ccf=' + e.ccf_group) : ''}${e.negated ? ' [NOT]' : ''}\n`;
  }
  document.getElementById('fta-out').textContent = s;
}
document.getElementById('f-load').addEventListener('click', () => currentAID && loadFTA(currentAID));
document.getElementById('g-type').addEventListener('change', e => {
  document.getElementById('g-k').style.display = (e.target.value === 'KOFN') ? '' : 'none';
});
document.getElementById('b-type').addEventListener('change', e => {
  document.getElementById('b-k').style.display = (e.target.value === 'kofn') ? '' : 'none';
});
document.getElementById('in-add').addEventListener('click', () => {
  const gid = document.getElementById('in-gate').value.trim();
  const ref = document.getElementById('in-ref').value.trim();
  if (!gid || !ref) return alert('请输入门ID与子门/事件ID');
  if (!inputs[gid]) inputs[gid] = [];
  if (gates.some(g => g.id === ref)) inputs[gid].push({ gate_id: ref });
  else if (events.some(e => e.id === ref)) inputs[gid].push({ event_id: ref });
  else return alert('子门/事件不存在，请先添加');
  renderFTA();
});
// add gate (uses g-id, g-type, g-k, g-top)
function addGateFromForm() {
  const id = document.getElementById('g-id').value.trim();
  if (!id) return alert('请输入门ID');
  if (gates.some(g => g.id === id)) return alert('门已存在');
  const type = document.getElementById('g-type').value;
  const k = parseInt(document.getElementById('g-k').value || '0', 10);
  const isTop = document.getElementById('g-top').checked;
  gates.push({ id, analysis_id: currentAID, type, k: type === 'KOFN' ? (k || 1) : 0, is_top: isTop, inputs: [] });
  if (isTop) gates.forEach(g => { g.is_top = (g.id === id); });
  inputs[id] = [];
  renderFTA();
}
// add event (uses e-id, e-label, e-prob, e-ccf, e-neg)
function addEventFromForm() {
  const id = document.getElementById('e-id').value.trim();
  if (!id) return alert('请输入事件ID');
  if (events.some(e => e.id === id)) return alert('事件已存在');
  const label = document.getElementById('e-label').value.trim();
  const prob = parseInt(document.getElementById('e-prob').value || '0', 10);
  const ccf = document.getElementById('e-ccf').value.trim();
  const neg = document.getElementById('e-neg').checked;
  events.push({ id, analysis_id: currentAID, label, prob_micro: prob, ccf_group: ccf, negated: neg });
  renderFTA();
}
// Bind "添加门"/"添加事件" — reuse in-add? We attach explicit handlers to the
// g-id enter key and e-id enter key as the add trigger.
document.getElementById('g-id').addEventListener('keydown', e => { if (e.key === 'Enter') { e.preventDefault(); addGateFromForm(); } });
document.getElementById('e-id').addEventListener('keydown', e => { if (e.key === 'Enter') { e.preventDefault(); addEventFromForm(); } });
document.getElementById('f-save').addEventListener('click', async () => {
  if (!currentAID) return alert('请先选择分析');
  const topGate = gates.find(g => g.is_top) || gates[0];
  if (!topGate) return alert('请先添加门');
  const payload = {
    analysis_id: currentAID,
    top_gate_id: topGate.id,
    gates: gates.map(g => ({ id: g.id, analysis_id: currentAID, type: g.type, k: g.k || 0, is_top: g.id === topGate.id, inputs: inputs[g.id] || [] })),
    events: events.map(e => ({ id: e.id, analysis_id: currentAID, label: e.label, component_id: e.component_id || '', prob_micro: e.prob_micro || 0, ccf_group: e.ccf_group || '', negated: !!e.negated })),
  };
  await j('POST', `/analyses/${currentAID}/fta`, payload);
  alert('故障树已保存');
});
document.getElementById('f-solve').addEventListener('click', async () => {
  if (!currentAID) return;
  const res = await j('POST', `/analyses/${currentAID}/solve`);
  document.getElementById('result-out').textContent = JSON.stringify(res, null, 2);
});

// --- FMEA ---
async function loadFMEA(id) {
  try {
    const tb = await j('GET', `/analyses/${id}/fmea`);
    const rows = (tb && tb.rows) || [];
    const body = document.querySelector('#m-table tbody');
    body.innerHTML = '';
    for (const r of rows) {
      const tr = document.createElement('tr');
      tr.innerHTML = `<td>${esc(r.function)}</td><td>${esc(r.failure_mode)}</td><td>${esc(r.effect)}</td><td>${r.severity}</td><td>${r.occurrence}</td><td>${r.detection}</td><td>${r.rpn}</td><td class="risk-${r.risk_class}">${r.risk_class}</td>`;
      body.appendChild(tr);
    }
  } catch (e) {}
}
document.getElementById('m-load').addEventListener('click', () => currentAID && loadFMEA(currentAID));
document.getElementById('m-add').addEventListener('click', async () => {
  if (!currentAID) return alert('请先选择分析');
  const row = {
    function: document.getElementById('m-fn').value.trim(),
    failure_mode: document.getElementById('m-mode').value.trim(),
    effect: document.getElementById('m-eff').value.trim(),
    severity: parseInt(document.getElementById('m-s').value || '0', 10),
    occurrence: parseInt(document.getElementById('m-o').value || '0', 10),
    detection: parseInt(document.getElementById('m-d').value || '0', 10),
  };
  await j('POST', `/analyses/${currentAID}/fmea/rows`, row);
  await loadFMEA(currentAID);
});

// --- RBD ---
let blocks = [], children = {};
function resetRBD() { blocks = []; children = {}; renderRBD(); }
async function loadRBD(id) {
  try {
    const d = await j('GET', `/analyses/${id}/rbd`);
    blocks = d.blocks || [];
    children = {};
    for (const b of blocks) { children[b.id] = b.children || []; }
    document.getElementById('r-root').value = d.root_id || '';
    renderRBD();
  } catch (e) { resetRBD(); }
}
function renderRBD() {
  let s = '块:\n';
  for (const b of blocks) {
    s += `  ${b.id} ${b.type}${b.k ? (' k=' + b.k) : ''}${b.failure_rate_ppt ? (' λ=' + b.failure_rate_ppt + 'ppt') : ''}${b.repair_rate_ppt ? (' μ=' + b.repair_rate_ppt + 'ppt') : ''} <- ${(children[b.id] || []).join(', ')}\n`;
  }
  document.getElementById('rbd-out').textContent = s;
}
document.getElementById('r-load').addEventListener('click', () => currentAID && loadRBD(currentAID));
document.getElementById('b-add').addEventListener('click', () => {
  const id = document.getElementById('b-id').value.trim();
  if (!id || blocks.some(b => b.id === id)) return alert('块ID缺失或已存在');
  const type = document.getElementById('b-type').value;
  const k = parseInt(document.getElementById('b-k').value || '0', 10);
  const rate = parseInt(document.getElementById('b-rate').value || '0', 10);
  const repair = parseInt(document.getElementById('b-repair').value || '0', 10);
  blocks.push({ id, analysis_id: currentAID, name: id, type, k: type === 'kofn' ? (k || 1) : 0, failure_rate_ppt: rate, repair_rate_ppt: repair });
  children[id] = [];
  renderRBD();
});
document.getElementById('c-add').addEventListener('click', () => {
  const bid = document.getElementById('c-block').value.trim();
  const cid = document.getElementById('c-child').value.trim();
  if (!bid || !cid || !blocks.some(b => b.id === bid)) return alert('父块ID缺失');
  if (!children[bid]) children[bid] = [];
  children[bid].push(cid);
  renderRBD();
});
document.getElementById('r-save').addEventListener('click', async () => {
  if (!currentAID) return;
  const root = document.getElementById('r-root').value.trim();
  if (!root) return alert('请输入根块ID');
  const payload = {
    analysis_id: currentAID, root_id: root,
    blocks: blocks.map(b => ({ id: b.id, analysis_id: currentAID, name: b.name, type: b.type, k: b.k || 0, failure_rate_ppt: b.failure_rate_ppt || 0, repair_rate_ppt: b.repair_rate_ppt || 0, children: children[b.id] || [] })),
  };
  await j('POST', `/analyses/${currentAID}/rbd`, payload);
  alert('可靠性框图已保存');
});
document.getElementById('r-solve').addEventListener('click', async () => {
  if (!currentAID) return;
  const res = await j('POST', `/analyses/${currentAID}/solve`);
  document.getElementById('rbd-out').textContent = JSON.stringify(res, null, 2);
});

// --- failure data ---
async function loadFDist(id) {
  try {
    const ev = await j('GET', `/analyses/${id}/failure-events`);
    const body = document.querySelector('#d-table tbody');
    body.innerHTML = '';
    for (const e of ev || []) {
      const tr = document.createElement('tr');
      tr.innerHTML = `<td>${e.ttf_hours}</td><td>${e.ttr_hours}</td>`;
      body.appendChild(tr);
    }
  } catch (e) {}
}
document.getElementById('d-load').addEventListener('click', () => currentAID && loadFDist(currentAID));
document.getElementById('d-add').addEventListener('click', async () => {
  if (!currentAID) return;
  const ev = {
    ttf_hours: parseInt(document.getElementById('d-ttf').value || '0', 10),
    ttr_hours: parseInt(document.getElementById('d-ttr').value || '0', 10),
  };
  await j('POST', `/analyses/${currentAID}/failure-events`, ev);
  await loadFDist(currentAID);
});
document.getElementById('d-fit').addEventListener('click', async () => {
  if (!currentAID) return;
  const fit = await j('GET', `/analyses/${currentAID}/failure-events/fit`);
  document.getElementById('fdist-out').textContent = JSON.stringify(fit, null, 2);
});

// --- result & lifecycle ---
document.getElementById('s-solve').addEventListener('click', async () => {
  if (!currentAID) return;
  const res = await j('POST', `/analyses/${currentAID}/solve`);
  document.getElementById('result-out').textContent = JSON.stringify(res, null, 2);
});
document.getElementById('s-get').addEventListener('click', async () => {
  if (!currentAID) return;
  const res = await j('GET', `/analyses/${currentAID}/report`);
  document.getElementById('result-out').textContent = JSON.stringify(res, null, 2);
});
async function lifecycle(path) {
  if (!currentAID) return;
  const a = await j('POST', `/analyses/${currentAID}/${path}`);
  await loadAnalyses();
  setAID(a.id); alert(`状态: ${a.state} v${a.version}`);
}
document.getElementById('s-review').addEventListener('click', () => lifecycle('review'));
document.getElementById('s-baseline').addEventListener('click', () => lifecycle('baseline'));
document.getElementById('s-revise').addEventListener('click', () => lifecycle('revise'));

// init
loadAnalyses();
