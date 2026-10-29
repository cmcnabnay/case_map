const COURSE_COLORS = {
  'Contracts': '#1a73e8',
  'Torts': '#d93025',
  'Civil Procedure': '#188038',
  'Lawyering Skills & Strategies': '#9334e6',
};
const OTHER_COLOR = '#5f6368';
// Pins where the opinion gives no precise place are drawn hollow; "forum" means
// the facts weren't localized, so the pin marks the court that heard the case.
const COARSE = ['state', 'country', 'forum'];

const map = new maplibregl.Map({
  container: 'map',
  style: 'https://tiles.openfreemap.org/styles/liberty',
  center: [-40, 30],
  zoom: 1.6,
  maxZoom: 22,
  renderWorldCopies: true,
  attributionControl: { compact: true },
});
map.addControl(new maplibregl.NavigationControl({ visualizePitch: true }), 'bottom-right');
map.addControl(new maplibregl.GeolocateControl({}), 'bottom-right');
map.addControl(new maplibregl.FullscreenControl(), 'bottom-right');
map.addControl(new maplibregl.ScaleControl({ unit: 'imperial' }), 'bottom-right');

let allCases = [];
const state = { query: '', courses: new Set() };

const esc = (s) => String(s ?? '').replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
const colorFor = (course) => COURSE_COLORS[course] || OTHER_COLOR;

function visibleCases() {
  const q = state.query.toLowerCase();
  return allCases.filter((c) =>
    state.courses.has(c.course) &&
    (!q || c.case.toLowerCase().includes(q) || (c.location?.place_name || '').toLowerCase().includes(q)));
}

function toGeoJSON(cases) {
  return {
    type: 'FeatureCollection',
    features: cases.filter((c) => c.location).map((c) => ({
      type: 'Feature',
      geometry: { type: 'Point', coordinates: [c.location.lng, c.location.lat] },
      properties: { name: c.case, color: colorFor(c.course), coarse: COARSE.includes(c.location.precision) },
    })),
  };
}

function addCaseLayers() {
  if (map.getSource('cases')) return;
  map.addSource('cases', { type: 'geojson', data: toGeoJSON(visibleCases()), cluster: true, clusterRadius: 40, clusterMaxZoom: 14 });
  map.addLayer({
    id: 'clusters', type: 'circle', source: 'cases', filter: ['has', 'point_count'],
    paint: {
      'circle-color': '#ffffff', 'circle-stroke-color': '#1a73e8', 'circle-stroke-width': 3,
      'circle-radius': ['step', ['get', 'point_count'], 14, 10, 18, 30, 24],
    },
  });
  map.addLayer({
    id: 'cluster-count', type: 'symbol', source: 'cases', filter: ['has', 'point_count'],
    layout: { 'text-field': ['get', 'point_count_abbreviated'], 'text-size': 12, 'text-font': ['Noto Sans Bold'] },
    paint: { 'text-color': '#1a73e8' },
  });
  map.addLayer({
    id: 'case-points', type: 'circle', source: 'cases', filter: ['!', ['has', 'point_count']],
    paint: {
      'circle-radius': ['interpolate', ['linear'], ['zoom'], 2, 6, 14, 9],
      'circle-color': ['case', ['get', 'coarse'], '#ffffff', ['get', 'color']],
      'circle-stroke-color': ['case', ['get', 'coarse'], ['get', 'color'], '#ffffff'],
      'circle-stroke-width': ['case', ['get', 'coarse'], 3, 2],
    },
  });
}

// Frame every located case, leaving room for the side panel on wide screens.
function fitToCases() {
  const pts = allCases.filter((c) => c.location);
  if (!pts.length) return;
  const bounds = new maplibregl.LngLatBounds();
  pts.forEach((c) => bounds.extend([c.location.lng, c.location.lat]));
  const wide = window.innerWidth > 600;
  map.fitBounds(bounds, { padding: { top: 40, bottom: 40, left: wide ? 400 : 20, right: 40 }, maxZoom: 4, duration: 0 });
}

function refresh() {
  const cases = visibleCases();
  map.getSource('cases')?.setData(toGeoJSON(cases));
  renderList(cases);
}

function renderList(cases) {
  const located = cases.filter((c) => c.location).length;
  document.getElementById('summary').textContent =
    `${cases.length} cases · ${located} on the map` + (located < cases.length ? ` · ${cases.length - located} not yet located` : '');
  const list = document.getElementById('list');
  list.innerHTML = '';
  for (const c of cases) {
    const li = document.createElement('li');
    if (!c.location) li.className = 'unlocated';
    li.innerHTML = `<span class="dot" style="background:${colorFor(c.course)}"></span>
      <span class="name">${esc(c.case)}</span>
      <span class="place">${esc(c.location?.place_name || 'Location not determined yet')}</span>`;
    if (c.location) li.addEventListener('click', () => focusCase(c));
    list.appendChild(li);
  }
}

const ZOOM_FOR = { address: 17, building: 17, town: 12, county: 9, region: 8, state: 6, country: 4, forum: 14 };

function focusCase(c) {
  const center = [c.location.lng, c.location.lat];
  map.flyTo({ center, zoom: ZOOM_FOR[c.location.precision] ?? 12, essential: true });
  map.once('moveend', () => openPopup(c));
  if (window.innerWidth <= 600) document.getElementById('panel').classList.add('collapsed');
}

let popup;
function openPopup(c) {
  popup?.remove();
  const l = c.location;
  const section = (title, text) => text ? `<h4>${title}</h4><p>${esc(text.replace(/^- /gm, ''))}</p>` : '';
  popup = new maplibregl.Popup({ maxWidth: '360px', offset: 12 })
    .setLngLat([l.lng, l.lat])
    .setHTML(`<div class="pop">
      <h3>${esc(c.case)}</h3>
      <div class="meta">${esc(c.course)}</div>
      <div class="body">
        <h4>Where</h4><p>${esc(l.place_name)}</p>
        ${l.address ? `<p class="addr">${esc(l.address)}</p>` : ''}
        ${l.evidence ? `<blockquote>${esc(l.evidence)}</blockquote>` : ''}
        ${section('Facts', c.facts)}
        ${section('Holding', c.holding)}
        ${l.notes ? section('Location notes', l.notes) : ''}
      </div></div>`)
    .addTo(map);
}

map.on('click', 'case-points', (e) => {
  const c = allCases.find((x) => x.case === e.features[0].properties.name);
  if (c) openPopup(c);
});
map.on('click', 'clusters', async (e) => {
  const f = e.features[0];
  const zoom = await map.getSource('cases').getClusterExpansionZoom(f.properties.cluster_id);
  map.easeTo({ center: f.geometry.coordinates, zoom });
});
for (const layer of ['case-points', 'clusters']) {
  map.on('mouseenter', layer, () => { map.getCanvas().style.cursor = 'pointer'; });
  map.on('mouseleave', layer, () => { map.getCanvas().style.cursor = ''; });
}

// Satellite: Esri imagery slipped under the labels, like Google's hybrid view.
function setBasemap(mode) {
  document.querySelectorAll('#basemap button').forEach((b) => b.classList.toggle('active', b.dataset.mode === mode));
  if (mode === 'satellite' && !map.getLayer('satellite')) {
    if (!map.getSource('satellite')) {
      map.addSource('satellite', {
        type: 'raster', tileSize: 256, maxzoom: 19,
        tiles: ['https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}'],
        attribution: 'Imagery © Esri, Maxar, Earthstar Geographics',
      });
    }
    const firstSymbol = map.getStyle().layers.find((l) => l.type === 'symbol')?.id;
    map.addLayer({ id: 'satellite', type: 'raster', source: 'satellite' }, firstSymbol);
  } else if (mode === 'map' && map.getLayer('satellite')) {
    map.removeLayer('satellite');
  }
}
document.querySelectorAll('#basemap button').forEach((b) => b.addEventListener('click', () => setBasemap(b.dataset.mode)));

function renderFilters() {
  const courses = [...new Set(allCases.map((c) => c.course))].sort();
  courses.forEach((c) => state.courses.add(c));
  const el = document.getElementById('filters');
  el.innerHTML = courses.map((c) =>
    `<label class="chip"><input type="checkbox" checked value="${esc(c)}"><span class="dot" style="background:${colorFor(c)}"></span>${esc(c)}</label>`).join('');
  el.addEventListener('change', (e) => {
    e.target.checked ? state.courses.add(e.target.value) : state.courses.delete(e.target.value);
    refresh();
  });
}

document.getElementById('search').addEventListener('input', (e) => { state.query = e.target.value; refresh(); });
document.getElementById('toggle-panel').addEventListener('click', () => document.getElementById('panel').classList.toggle('collapsed'));

// Show the list as soon as the data arrives; pins are added once the map style is ready.
const casesReady = fetch('/api/cases')
  .then((r) => { if (!r.ok) throw new Error(r.statusText); return r.json(); })
  .then((cases) => {
    allCases = cases;
    renderFilters();
    refresh();
    return cases;
  })
  .catch((err) => {
    document.getElementById('summary').textContent = `Could not load cases: ${err.message}`;
    throw err;
  });

map.on('load', () => {
  casesReady.then(() => {
    addCaseLayers();
    refresh();
    fitToCases();
  }, () => {});
});
