'use strict';
/* =====================================================================
   SPADES - A Brooklyn Rooftop Card Game
   All card art and props are drawn here in code (original artwork, (c) Ironite).
   ===================================================================== */
const Art = (() => {
  const CW = 360, CH = 504;              // card canvas size (HD; the game scales it down)
  const INK = '#24170f';
  const RED = '#a01c30', BLACK = '#18141a', GOLD = '#b8913f', GOLD2 = '#e8cf8a';
  const IVORY1 = '#fdf9ee', IVORY2 = '#f1e6cc';
  const SERIF = '"Rubik", Arial, sans-serif';
  const DISP = '"Oswald", Impact, sans-serif';
  const SCRIPT = '"Pacifico", cursive';
  const TAU = Math.PI * 2;

  // Suit palettes for the court cards: brick, sunset amber, night navy, park green.
  const PAL = {
    H: { coat: '#b23a2a', dark: '#6e1e14', accent: '#e8c35a', ink: RED, wall: '#ecc9bc' },
    D: { coat: '#d9902e', dark: '#7e4a14', accent: '#f6efe0', ink: RED, wall: '#f2d9b8' },
    S: { coat: '#1f2a4a', dark: '#0e1428', accent: '#e8c35a', ink: BLACK, wall: '#d4d8e6' },
    C: { coat: '#2f5a44', dark: '#183426', accent: '#e8c35a', ink: BLACK, wall: '#d4e2d0' }
  };

  /* ---------- small helpers ---------- */
  function rng(seed) { let s = seed >>> 0 || 1; return () => (s = (s * 1664525 + 1013904223) >>> 0) / 4294967296; }
  function rrect(c, x, y, w, h, r) { c.beginPath(); c.moveTo(x + r, y); c.arcTo(x + w, y, x + w, y + h, r); c.arcTo(x + w, y + h, x, y + h, r); c.arcTo(x, y + h, x, y, r); c.arcTo(x, y, x + w, y, r); c.closePath(); }
  function canvas(w, h) { const c = document.createElement('canvas'); c.width = w; c.height = h; return c; }
  const ink = c => { c.strokeStyle = INK; c.lineWidth = 2.6; c.lineJoin = 'round'; c.lineCap = 'round'; };

  /* ---------- suit shapes (drawn, so they look the same on every PC) ---------- */
  function heartPath(c, x, y, s) {
    c.moveTo(x, y + s * 0.45);
    c.bezierCurveTo(x - s * 0.62, y + s * 0.02, x - s * 0.52, y - s * 0.55, x, y - s * 0.2);
    c.bezierCurveTo(x + s * 0.52, y - s * 0.55, x + s * 0.62, y + s * 0.02, x, y + s * 0.45);
  }
  function suitPath(c, s, x, y, size) {
    if (s === 'H') heartPath(c, x, y, size);
    else if (s === 'D') { c.moveTo(x, y - size * 0.5); c.quadraticCurveTo(x + size * 0.18, y - size * 0.18, x + size * 0.4, y); c.quadraticCurveTo(x + size * 0.18, y + size * 0.18, x, y + size * 0.5); c.quadraticCurveTo(x - size * 0.18, y + size * 0.18, x - size * 0.4, y); c.quadraticCurveTo(x - size * 0.18, y - size * 0.18, x, y - size * 0.5); }
    else if (s === 'S') {
      c.save(); c.translate(x, y - size * 0.08); c.rotate(Math.PI); heartPath(c, 0, 0, size * 0.9); c.restore();
      c.moveTo(x, y + size * 0.05); c.quadraticCurveTo(x - size * 0.05, y + size * 0.38, x - size * 0.24, y + size * 0.5); c.lineTo(x + size * 0.24, y + size * 0.5); c.quadraticCurveTo(x + size * 0.05, y + size * 0.38, x, y + size * 0.05);
    } else {
      const r = size * 0.205;
      c.moveTo(x + r, y - size * 0.24); c.arc(x, y - size * 0.24, r, 0, TAU); c.moveTo(x - size * 0.22 + r, y + size * 0.06); c.arc(x - size * 0.22, y + size * 0.06, r, 0, TAU);
      c.moveTo(x + size * 0.22 + r, y + size * 0.06); c.arc(x + size * 0.22, y + size * 0.06, r, 0, TAU);
      c.moveTo(x + size * 0.12, y); c.arc(x, y, size * 0.12, 0, TAU);
      c.moveTo(x, y + size * 0.02); c.quadraticCurveTo(x - size * 0.04, y + size * 0.38, x - size * 0.22, y + size * 0.5); c.lineTo(x + size * 0.22, y + size * 0.5); c.quadraticCurveTo(x + size * 0.04, y + size * 0.38, x, y + size * 0.02);
    }
  }
  function suit(c, s, x, y, size, color, shine = true) {
    c.save(); c.fillStyle = color; c.beginPath(); suitPath(c, s, x, y, size); c.fill();
    if (shine && size > 30) { // a little lacquer highlight
      c.clip(); c.fillStyle = 'rgba(255,255,255,.16)'; c.beginPath(); c.ellipse(x - size * 0.16, y - size * 0.16, size * 0.12, size * 0.2, -0.6, 0, TAU); c.fill();
    }
    c.restore();
  }

  /* ---------- paper ---------- */
  function paper(c, w, h, seed, r = 22) {
    const g = c.createLinearGradient(0, 0, w, h); g.addColorStop(0, IVORY1); g.addColorStop(1, IVORY2);
    rrect(c, 1, 1, w - 2, h - 2, r); c.fillStyle = g; c.fill();
    c.save(); c.clip();
    const R = rng(seed);
    for (let i = 0; i < 900; i++) { c.fillStyle = `rgba(120,90,40,${R() * 0.045})`; c.fillRect(R() * w, R() * h, 1 + R() * 2, 1 + R() * 2); }
    const v = c.createRadialGradient(w / 2, h / 2, h * 0.3, w / 2, h / 2, h * 0.75); v.addColorStop(0, 'rgba(0,0,0,0)'); v.addColorStop(1, 'rgba(120,80,30,.14)');
    c.fillStyle = v; c.fillRect(0, 0, w, h); c.restore();
    rrect(c, 1, 1, w - 2, h - 2, r); c.strokeStyle = 'rgba(70,50,30,.45)'; c.lineWidth = 2; c.stroke();
  }
  // an Art Deco border: a double line with stepped corners and a little spade
  function border(c, w, h, color) {
    c.save(); c.strokeStyle = color; c.fillStyle = color; c.globalAlpha = 0.85;
    c.lineWidth = 2; rrect(c, 14, 14, w - 28, h - 28, 8); c.stroke();
    c.lineWidth = 0.8; rrect(c, 20, 20, w - 40, h - 40, 5); c.stroke();
    for (const [x, y, sx, sy] of [[20, 20, 1, 1], [w - 20, 20, -1, 1], [20, h - 20, 1, -1], [w - 20, h - 20, -1, -1]]) {
      c.lineWidth = 1.4;
      c.beginPath(); c.moveTo(x + sx * 6, y + sy * 40); c.lineTo(x + sx * 6, y + sy * 6); c.lineTo(x + sx * 40, y + sy * 6); c.stroke();
      c.beginPath(); c.moveTo(x + sx * 12, y + sy * 26); c.lineTo(x + sx * 12, y + sy * 12); c.lineTo(x + sx * 26, y + sy * 12); c.stroke();
      c.beginPath(); c.moveTo(x + sx * 19, y + sy * 14); c.lineTo(x + sx * 24, y + sy * 19); c.lineTo(x + sx * 19, y + sy * 24); c.lineTo(x + sx * 14, y + sy * 19); c.closePath(); c.fill();
    }
    c.restore();
  }
  function index(c, rank, s, color, w, h) {
    const draw = () => {
      c.fillStyle = color; c.textAlign = 'center'; c.textBaseline = 'alphabetic';
      c.font = `700 ${rank === '10' ? 46 : 54}px ${DISP}`;
      c.fillText(rank, 46, 74);
      suit(c, s, 46, 102, 32, color, false);
    };
    draw(); c.save(); c.translate(w, h); c.rotate(Math.PI); draw(); c.restore();
  }

  /* ---------- the neighborhood people (court figures) ----------
     Drawn around (0,0) = bottom centre of the bust. Head is ~54px wide at scale 1. */
  function shoulderX(y) {
    if (y <= -64) return 22;
    const bx = t => { const u = 1 - t; return [u * u * u * 22 + 3 * u * u * t * 54 + 3 * u * t * t * 78 + t * t * t * 80, u * u * u * -64 + 3 * u * u * t * -60 + 3 * u * t * t * -42]; };
    let lo = 0, hi = 1; for (let i = 0; i < 30; i++) { const m = (lo + hi) / 2; if (bx(m)[1] < y) lo = m; else hi = m; } return bx(lo)[0];
  }
  function shoulders(c) { c.beginPath(); c.moveTo(-80, 0); c.bezierCurveTo(-78, -42, -54, -60, -22, -64); c.lineTo(22, -64); c.bezierCurveTo(54, -60, 78, -42, 80, 0); c.closePath(); }
  function bust(c, o) {
    const skin = o.skin || '#f2d4b6', hair = o.hair || '#4a2e1c', hy = -118;
    c.save(); ink(c);
    c.fillStyle = hair;
    if (o.hairStyle === 'bun') { c.beginPath(); c.arc(0, hy - 36, 15, 0, TAU); c.fill(); c.stroke(); }
    if (o.hairStyle === 'waves') { c.beginPath(); c.moveTo(-30, hy - 10); c.bezierCurveTo(-44, hy + 20, -40, hy + 52, -26, hy + 60); c.lineTo(26, hy + 60); c.bezierCurveTo(40, hy + 52, 44, hy + 20, 30, hy - 10); c.closePath(); c.fill(); c.stroke(); }
    if (o.hairStyle === 'bob') { c.beginPath(); c.moveTo(-31, hy - 8); c.bezierCurveTo(-38, hy + 14, -36, hy + 28, -30, hy + 32); c.lineTo(30, hy + 32); c.bezierCurveTo(36, hy + 28, 38, hy + 14, 31, hy - 8); c.closePath(); c.fill(); c.stroke(); }
    // the top
    const top = (o.top === 'stripes' || o.top === 'chef' || o.top === 'baker') ? '#fbf8f0' : o.coat;
    shoulders(c); c.fillStyle = top; c.fill();
    if (o.top === 'stripes') { c.fillStyle = o.stripe || '#1f2f55'; for (let y = -58; y < 2; y += 13) { const y0 = y, y1 = Math.min(y + 6, 0), x0 = shoulderX(y0), x1 = shoulderX(y1); c.beginPath(); c.moveTo(-x0, y0); c.lineTo(x0, y0); c.lineTo(x1, y1); c.lineTo(-x1, y1); c.closePath(); c.fill(); } }
    if (o.top === 'dress') { c.save(); shoulders(c); c.clip(); c.fillStyle = 'rgba(255,255,255,.12)'; for (let x = -80; x < 80; x += 14) { c.beginPath(); c.moveTo(x, 0); c.quadraticCurveTo(x + 6, -40, x + 2, -70); c.lineTo(x + 6, -70); c.quadraticCurveTo(x + 10, -40, x + 4, 0); c.fill(); } c.restore(); }
    shoulders(c); c.stroke();
    // neck
    c.fillStyle = skin; c.beginPath(); c.moveTo(-11, -90); c.lineTo(-12, -61); c.lineTo(12, -61); c.lineTo(11, -90); c.closePath(); c.fill();
    c.fillStyle = 'rgba(120,60,40,.18)'; c.beginPath(); c.moveTo(-11, -88); c.quadraticCurveTo(0, -80, 11, -88); c.lineTo(11, -84); c.quadraticCurveTo(0, -76, -11, -84); c.closePath(); c.fill();
    if (o.top === 'dress') { // a sweetheart neckline
      c.fillStyle = skin; c.beginPath(); c.moveTo(-26, -64); c.quadraticCurveTo(-14, -42, 0, -50); c.quadraticCurveTo(14, -42, 26, -64); c.closePath(); c.fill();
      c.beginPath(); c.moveTo(-26, -64); c.quadraticCurveTo(-14, -42, 0, -50); c.quadraticCurveTo(14, -42, 26, -64); c.stroke();
    }
    if (o.top === 'stripes') { c.beginPath(); c.moveTo(-23, -64); c.quadraticCurveTo(0, -55, 23, -64); c.stroke(); }
    if (o.top === 'blouse') { c.fillStyle = o.accent; for (const sx of [-1, 1]) { c.beginPath(); c.moveTo(sx * 2, -62); c.lineTo(sx * 20, -66); c.lineTo(sx * 16, -50); c.closePath(); c.fill(); c.stroke(); } }
    if (o.top === 'turtleneck') { c.fillStyle = o.coat; rrect(c, -15, -82, 30, 22, 6); c.fill(); c.stroke(); c.strokeStyle = o.dark; c.lineWidth = 1.2; for (let x = -10; x <= 10; x += 5) { c.beginPath(); c.moveTo(x, -80); c.lineTo(x, -62); c.stroke(); } ink(c); }
    if (o.top === 'suit' || o.top === 'waistcoat') {
      c.fillStyle = '#fbf8f0'; c.beginPath(); c.moveTo(-18, -64); c.lineTo(0, -14); c.lineTo(18, -64); c.closePath(); c.fill(); c.stroke();
      if (o.top === 'waistcoat') { c.fillStyle = o.vest || '#1e1e24'; for (const sx of [-1, 1]) { c.beginPath(); c.moveTo(sx * 10, -48); c.lineTo(sx * 2, -8); c.lineTo(sx * 2, 0); c.lineTo(sx * 40, 0); c.lineTo(sx * 34, -50); c.closePath(); c.fill(); c.stroke(); } c.fillStyle = GOLD; for (const y of [-30, -18, -6]) { c.beginPath(); c.arc(6, y, 2.4, 0, TAU); c.fill(); } }
      c.fillStyle = o.dark; for (const sx of [-1, 1]) { c.beginPath(); c.moveTo(sx * 20, -64); c.lineTo(sx * (o.top === 'waistcoat' ? 34 : 3), -14); c.lineTo(sx * 46, -34); c.lineTo(sx * 30, -54); c.closePath(); c.fill(); c.stroke(); }
      c.fillStyle = '#fbf8f0'; for (const sx of [-1, 1]) { c.beginPath(); c.moveTo(sx * 1, -62); c.lineTo(sx * 13, -66); c.lineTo(sx * 10, -54); c.closePath(); c.fill(); c.stroke(); }
    }
    if (o.top === 'cardigan') {
      c.fillStyle = o.accent; c.beginPath(); c.moveTo(-18, -64); c.lineTo(0, -26); c.lineTo(18, -64); c.closePath(); c.fill(); c.stroke();
      c.fillStyle = o.dark; for (const y of [-18, -6]) { c.beginPath(); c.arc(0, y, 3, 0, TAU); c.fill(); }
    }
    if (o.top === 'chef' || o.top === 'baker') {
      c.fillStyle = '#d8d4cc'; for (const y of [-46, -30, -14]) for (const x of [-14, 14]) { c.beginPath(); c.arc(x, y, 3.2, 0, TAU); c.fill(); c.stroke(); }
      if (o.top === 'baker') { c.fillStyle = '#e9dcc0'; c.beginPath(); c.moveTo(-34, 0); c.lineTo(-30, -38); c.lineTo(30, -38); c.lineTo(34, 0); c.closePath(); c.fill(); c.stroke(); c.fillStyle = 'rgba(255,255,255,.7)'; c.beginPath(); c.arc(-14, -24, 4, 0, TAU); c.arc(10, -14, 3, 0, TAU); c.fill(); }
    }
    // neckwear
    if (o.neck === 'scarf') {
      c.fillStyle = o.neckColor; c.beginPath(); c.moveTo(-17, -78); c.quadraticCurveTo(0, -66, 17, -78); c.lineTo(19, -66); c.quadraticCurveTo(0, -54, -19, -66); c.closePath(); c.fill(); c.stroke();
      c.beginPath(); c.moveTo(8, -64); c.quadraticCurveTo(20, -46, 14, -26); c.lineTo(24, -30); c.quadraticCurveTo(26, -48, 14, -62); c.closePath(); c.fill(); c.stroke();
      c.beginPath(); c.ellipse(9, -63, 6, 5, 0, 0, TAU); c.fill(); c.stroke();
    }
    if (o.neck === 'pearls') { c.fillStyle = '#fffaf0'; c.lineWidth = 1.2; for (let i = 0; i < 11; i++) { const a = Math.PI * (0.12 + i * 0.076); c.beginPath(); c.arc(Math.cos(a) * 22, -70 + Math.sin(a) * 16, 3, 0, TAU); c.fill(); c.stroke(); } ink(c); }
    if (o.neck === 'bowtie') { c.fillStyle = o.neckColor; c.beginPath(); c.moveTo(0, -60); c.lineTo(-13, -67); c.lineTo(-13, -52); c.closePath(); c.moveTo(0, -60); c.lineTo(13, -67); c.lineTo(13, -52); c.closePath(); c.fill(); c.stroke(); rrect(c, -3.5, -63.5, 7, 7, 2); c.fill(); c.stroke(); }
    if (o.neck === 'kerchief') { c.fillStyle = o.neckColor; c.beginPath(); c.moveTo(-16, -66); c.quadraticCurveTo(0, -58, 16, -66); c.lineTo(0, -44); c.closePath(); c.fill(); c.stroke(); }
    if (o.neck === 'choker') { c.fillStyle = '#18141a'; c.fillRect(-11, -84, 22, 5); c.fillStyle = '#9a7ab8'; c.beginPath(); c.arc(0, -79, 3.5, 0, TAU); c.fill(); c.stroke(); }
    // head
    c.fillStyle = skin; c.beginPath(); c.ellipse(-27, hy + 2, 5, 8, 0, 0, TAU); c.ellipse(27, hy + 2, 5, 8, 0, 0, TAU); c.fill(); c.stroke();
    if (o.earrings) { c.fillStyle = o.earrings; for (const sx of [-1, 1]) { c.beginPath(); c.arc(sx * 28, hy + 13, 3.4, 0, TAU); c.fill(); } }
    c.fillStyle = skin; c.beginPath(); c.ellipse(0, hy, 27, 33, 0, 0, TAU); c.fill(); c.stroke();
    // front hair
    c.fillStyle = hair;
    if (o.hairStyle === 'short') { c.beginPath(); c.ellipse(0, hy - 14, 28.5, 22, 0, Math.PI, 0); c.quadraticCurveTo(14, hy - 22, -4, hy - 18); c.quadraticCurveTo(-18, hy - 14, -28.5, hy - 6); c.closePath(); c.fill(); c.stroke(); }
    if (o.hairStyle === 'side') { c.beginPath(); c.moveTo(-28.5, hy - 2); c.bezierCurveTo(-34, hy - 36, -6, hy - 44, 6, hy - 38); c.bezierCurveTo(24, hy - 40, 34, hy - 24, 28.5, hy - 4); c.quadraticCurveTo(24, hy - 22, 8, hy - 24); c.quadraticCurveTo(-2, hy - 26, -10, hy - 20); c.quadraticCurveTo(-20, hy - 14, -28.5, hy - 2); c.closePath(); c.fill(); c.stroke(); c.strokeStyle = 'rgba(255,255,255,.22)'; c.lineWidth = 1.4; c.beginPath(); c.moveTo(-12, hy - 32); c.quadraticCurveTo(4, hy - 38, 18, hy - 32); c.stroke(); ink(c); }
    if (o.hairStyle === 'bob') { c.beginPath(); c.moveTo(-29, hy + 14); c.bezierCurveTo(-36, hy - 34, 36, hy - 34, 29, hy + 14); c.lineTo(26, hy - 6); c.quadraticCurveTo(18, hy - 20, 4, hy - 16); c.quadraticCurveTo(-10, hy - 20, -20, hy - 14); c.quadraticCurveTo(-26, hy - 8, -29, hy + 14); c.closePath(); c.fill(); c.stroke(); }
    if (o.hairStyle === 'bun' || o.hairStyle === 'waves') { c.beginPath(); c.moveTo(-29, hy + 6); c.bezierCurveTo(-34, hy - 30, -14, hy - 40, 0, hy - 36); c.bezierCurveTo(14, hy - 40, 34, hy - 30, 29, hy + 6); c.bezierCurveTo(22, hy - 10, 14, hy - 24, 0, hy - 22); c.bezierCurveTo(-14, hy - 24, -22, hy - 10, -29, hy + 6); c.closePath(); c.fill(); c.stroke(); }
    if (o.beard) { c.fillStyle = o.beardColor || hair; c.beginPath(); c.moveTo(-25, hy + 4); c.quadraticCurveTo(-24, hy + 42, 0, hy + 48); c.quadraticCurveTo(24, hy + 42, 25, hy + 4); c.quadraticCurveTo(14, hy + 22, 0, hy + 20); c.quadraticCurveTo(-14, hy + 22, -25, hy + 4); c.fill(); c.stroke(); }
    // face
    c.fillStyle = 'rgba(214,110,100,.28)'; c.beginPath(); c.arc(-15, hy + 9, 6.5, 0, TAU); c.arc(15, hy + 9, 6.5, 0, TAU); c.fill();
    c.fillStyle = INK; const ey = o.lady ? 3.4 : 3;
    c.beginPath(); c.ellipse(-10, hy - 3, 2.6, ey, 0, 0, TAU); c.ellipse(10, hy - 3, 2.6, ey, 0, 0, TAU); c.fill();
    if (o.lady) { c.lineWidth = 1.4; for (const sx of [-1, 1]) { c.beginPath(); c.moveTo(sx * 12.5, hy - 5); c.lineTo(sx * 15.5, hy - 7); c.stroke(); } }
    c.lineWidth = 2.2; c.strokeStyle = o.brow || hair; const st = o.stern ? 2 : 0, st1 = o.stern ? 0 : 1;
    c.beginPath(); c.moveTo(-15, hy - 11 + st); c.lineTo(-5, hy - 12 - st1); c.moveTo(15, hy - 11 + st); c.lineTo(5, hy - 12 - st1); c.stroke();
    c.strokeStyle = INK; c.lineWidth = 1.8; c.beginPath(); c.moveTo(1, hy); c.quadraticCurveTo(-3, hy + 9, 1, hy + 11); c.stroke();
    if (o.mole) { c.fillStyle = INK; c.beginPath(); c.arc(-13, hy + 16, 1.5, 0, TAU); c.fill(); }
    if (o.lady) { c.fillStyle = o.lip || '#b3283a'; c.beginPath(); if (o.smile) { c.moveTo(-7, hy + 18); c.quadraticCurveTo(0, hy + 26, 7, hy + 18); c.quadraticCurveTo(0, hy + 21, -7, hy + 18); } else c.ellipse(0, hy + 20, 6, o.stern ? 1.8 : 2.8, 0, 0, TAU); c.fill(); }
    else if (!o.mustache || o.mustache === 'pencil') { c.beginPath(); if (o.smile) { c.moveTo(-7, hy + 18); c.quadraticCurveTo(0, hy + 24, 7, hy + 18); } else { c.moveTo(-6, hy + 20); c.quadraticCurveTo(0, hy + 22, 6, hy + 20); } c.stroke(); }
    const mc = o.mustacheColor || o.brow || hair;
    if (o.mustache === 'pencil') { c.strokeStyle = mc; c.lineWidth = 2.2; c.beginPath(); c.moveTo(-10, hy + 14); c.quadraticCurveTo(0, hy + 11, 10, hy + 14); c.stroke(); }
    if (o.mustache === 'handlebar') { c.fillStyle = mc; c.strokeStyle = INK; c.lineWidth = 1.6; c.beginPath(); c.moveTo(0, hy + 13); c.bezierCurveTo(-8, hy + 10, -18, hy + 18, -24, hy + 10); c.bezierCurveTo(-28, hy + 6, -26, hy + 2, -22, hy + 4); c.bezierCurveTo(-22, hy + 10, -16, hy + 20, 0, hy + 17); c.bezierCurveTo(16, hy + 20, 22, hy + 10, 22, hy + 4); c.bezierCurveTo(26, hy + 2, 28, hy + 6, 24, hy + 10); c.bezierCurveTo(18, hy + 18, 8, hy + 10, 0, hy + 13); c.fill(); c.stroke(); }
    if (o.mustache === 'walrus') { c.fillStyle = mc; c.strokeStyle = INK; c.lineWidth = 1.6; c.beginPath(); c.moveTo(0, hy + 11); c.bezierCurveTo(-10, hy + 8, -18, hy + 12, -18, hy + 22); c.quadraticCurveTo(-8, hy + 18, 0, hy + 20); c.quadraticCurveTo(8, hy + 18, 18, hy + 22); c.bezierCurveTo(18, hy + 12, 10, hy + 8, 0, hy + 11); c.fill(); c.stroke(); }
    ink(c);
    if (o.glasses === 'round') { c.strokeStyle = '#3a2a1e'; c.lineWidth = 2; c.beginPath(); c.arc(-10, hy - 3, 7.5, 0, TAU); c.moveTo(17.5, hy - 3); c.arc(10, hy - 3, 7.5, 0, TAU); c.moveTo(-2.5, hy - 4); c.lineTo(2.5, hy - 4); c.stroke(); ink(c); }
    // hats
    if (o.hat === 'beret') { c.save(); c.translate(4, hy - 30); c.rotate(-0.14); c.fillStyle = o.hatColor; c.beginPath(); c.ellipse(0, 0, 36, 13, 0, 0, TAU); c.fill(); c.stroke(); c.beginPath(); c.moveTo(-2, -12); c.lineTo(0, -20); c.lineTo(3, -12); c.closePath(); c.fill(); c.stroke(); c.restore(); }
    if (o.hat === 'cloche') {
      c.fillStyle = o.hatColor; c.beginPath(); c.moveTo(-33, hy - 4); c.bezierCurveTo(-36, hy - 50, 36, hy - 50, 33, hy - 4); c.quadraticCurveTo(0, hy - 16, -33, hy - 4); c.closePath(); c.fill(); c.stroke();
      c.strokeStyle = o.band || 'rgba(255,255,255,.4)'; c.lineWidth = 4; c.beginPath(); c.moveTo(-31, hy - 14); c.quadraticCurveTo(0, hy - 26, 31, hy - 14); c.stroke(); ink(c);
      if (o.flowers) { for (const [x, y, col] of [[20, hy - 22, '#e8b4b8'], [27, hy - 17, '#f0d36a'], [21, hy - 13, '#e8b4b8']]) { c.fillStyle = col; c.beginPath(); c.arc(x, y, 4.5, 0, TAU); c.fill(); c.stroke(); } }
      if (o.veil) { c.save(); c.globalAlpha = 0.55; c.strokeStyle = '#18141a'; c.lineWidth = 0.9; c.beginPath(); c.moveTo(-31, hy - 10); c.quadraticCurveTo(0, hy + 14, 31, hy - 10); c.stroke(); for (let x = -28; x <= 28; x += 5) { c.beginPath(); c.moveTo(x, hy - 12); c.lineTo(x * 0.8, hy + 6 - Math.abs(x) * 0.3); c.stroke(); } for (let y = hy - 8; y < hy + 6; y += 5) { c.beginPath(); c.moveTo(-28 + (y - hy) * 0.3, y); c.lineTo(28 - (y - hy) * 0.3, y); c.stroke(); } c.restore(); ink(c); }
    }
    if (o.hat === 'cap') { c.fillStyle = o.hatColor; c.beginPath(); c.moveTo(-30, hy - 14); c.bezierCurveTo(-30, hy - 42, 26, hy - 46, 32, hy - 22); c.quadraticCurveTo(4, hy - 26, -30, hy - 14); c.closePath(); c.fill(); c.stroke(); c.beginPath(); c.moveTo(4, hy - 24); c.quadraticCurveTo(34, hy - 28, 40, hy - 16); c.quadraticCurveTo(24, hy - 12, 4, hy - 18); c.closePath(); c.fill(); c.stroke(); }
    if (o.hat === 'fedora') {
      c.fillStyle = o.hatColor; c.beginPath(); c.ellipse(0, hy - 22, 46, 10, 0, 0, TAU); c.fill(); c.stroke();
      c.beginPath(); c.moveTo(-26, hy - 24); c.bezierCurveTo(-28, hy - 60, -10, hy - 62, 0, hy - 54); c.bezierCurveTo(10, hy - 62, 28, hy - 60, 26, hy - 24); c.closePath(); c.fill(); c.stroke();
      c.strokeStyle = o.band || '#b23a2a'; c.lineWidth = 6; c.beginPath(); c.moveTo(-26, hy - 30); c.quadraticCurveTo(0, hy - 34, 26, hy - 30); c.stroke(); ink(c);
    }
    if (o.hat === 'toque' || o.hat === 'bakercap') {
      c.fillStyle = '#fdfcf8'; rrect(c, -24, hy - 46, 48, 16, 3); c.fill(); c.stroke();
      const puffs = o.hat === 'toque' ? [[-18, hy - 60, 15], [18, hy - 60, 15], [0, hy - 70, 18]] : [[-10, hy - 50, 12], [10, hy - 50, 12]];
      for (const [x, y, r] of puffs) { c.beginPath(); c.arc(x, y, r, 0, TAU); c.fill(); c.stroke(); }
      c.fillRect(-22, o.hat === 'toque' ? hy - 58 : hy - 50, 44, o.hat === 'toque' ? 14 : 6);
    }
    if (o.hat === 'feathers') {   // a cabaret headband with a plume
      c.strokeStyle = o.band || '#e8c35a'; c.lineWidth = 4; c.beginPath(); c.moveTo(-29, hy - 14); c.quadraticCurveTo(0, hy - 30, 29, hy - 14); c.stroke(); ink(c);
      c.fillStyle = '#e8c35a'; c.beginPath(); c.arc(16, hy - 22, 5, 0, TAU); c.fill(); c.stroke();
      c.fillStyle = o.plume || '#f6efe2'; c.beginPath(); c.moveTo(16, hy - 26); c.bezierCurveTo(24, hy - 70, 52, hy - 74, 48, hy - 50); c.bezierCurveTo(40, hy - 58, 30, hy - 44, 16, hy - 26); c.fill(); c.stroke();
    }
    c.restore();
  }
  function hand(c, x, y, skin) { c.save(); ink(c); c.fillStyle = skin || '#f2d4b6'; c.beginPath(); c.ellipse(x, y, 9, 8, 0, 0, TAU); c.fill(); c.stroke(); c.restore(); }

  // who sits on each court card
  const COURT = {
    K: {
      H: { name: 'The Pizza Man', o: { top: 'chef', hat: 'toque', neck: 'kerchief', neckColor: '#b23a2a', hair: '#8a8a90', mustache: 'handlebar', mustacheColor: '#d8d4cc', brow: '#c8c4c0', skin: '#eec3a2', smile: true }, prop: 'pizza' },
      D: { name: 'The Bodega Man', o: { top: 'cardigan', coat: '#c7822a', dark: '#7e4a14', accent: '#f4ecd8', hat: 'cap', hatColor: '#2a3a5c', mustache: 'walrus', mustacheColor: '#3a2a22', hair: '#2a1e18', glasses: 'round', skin: '#c98f66', smile: true }, prop: 'cat' },
      S: { name: 'The Jazzman', o: { top: 'suit', coat: '#1f2a4a', dark: '#0e1428', hat: 'fedora', hatColor: '#1a1a22', band: '#b23a2a', neck: 'bowtie', neckColor: '#e8c35a', mustache: 'pencil', hair: '#1c1414', skin: '#8a5a3a' }, prop: 'sax' },
      C: { name: 'The Super', o: { top: 'cardigan', coat: '#2f5a44', dark: '#183426', accent: '#f4ecd8', hair: '#a8a4a0', hairStyle: 'short', beard: true, beardColor: '#c8c4c0', brow: '#b8b4b0', skin: '#e3b48e' }, prop: 'keys' }
    },
    Q: {
      H: { name: 'The Singer', o: { lady: true, top: 'dress', coat: '#b23a2a', dark: '#6e1e14', hairStyle: 'waves', hair: '#1c1414', neck: 'pearls', lip: '#8a1a2a', skin: '#a0684a', earrings: '#e8c35a', smile: true }, prop: 'mic' },
      D: { name: 'The Florist', o: { lady: true, top: 'blouse', coat: '#d9902e', dark: '#8a5214', accent: '#f6efe0', hairStyle: 'bob', hair: '#b5651d', hat: 'cloche', hatColor: '#e9d9b0', band: '#c7822a', flowers: true, neck: 'scarf', neckColor: '#3a7a9a', skin: '#f2d4b6', smile: true }, prop: 'bouquet' },
      S: { name: 'The DJ', o: { lady: true, top: 'turtleneck', coat: '#2a1830', dark: '#120a18', hairStyle: 'bun', hair: '#120c10', lip: '#9c2a40', skin: '#6a4430', earrings: '#d8dce4', smile: true }, prop: 'record' },
      C: { name: 'The Muralist', o: { lady: true, top: 'blouse', coat: '#2f5a44', dark: '#183426', accent: '#fbf8f0', hairStyle: 'bob', hair: '#6b3f22', hat: 'beret', hatColor: '#b23a2a', neck: 'scarf', neckColor: '#e8b04a', skin: '#d6a27a', earrings: '#e8c35a', smile: true }, prop: 'spray' }
    },
    J: {
      H: { name: 'The Delivery Kid', o: { top: 'cardigan', coat: '#b23a2a', dark: '#6e1e14', accent: '#f4ecd8', hat: 'cap', hatColor: '#b23a2a', hair: '#4a2e1c', hairStyle: 'short', skin: '#f2d4b6', smile: true }, prop: 'boxes' },
      D: { name: 'The Bagel Man', o: { top: 'baker', hat: 'bakercap', neck: 'kerchief', neckColor: '#c7822a', hair: '#b5651d', skin: '#f6dbc4', smile: true, hairStyle: 'short' }, prop: 'bagels' },
      S: { name: 'The Slugger', o: { top: 'stripes', stripe: '#1f2a4a', hat: 'cap', hatColor: '#1f2a4a', hair: '#1c1414', hairStyle: 'short', skin: '#8a5a3a', smile: true }, prop: 'bat' },
      C: { name: 'The Newsie', o: { top: 'suit', coat: '#5a4a3a', dark: '#3a2e22', hat: 'cap', hatColor: '#4a3a2a', neck: 'tie', neckColor: '#e8c35a', hair: '#8a5a30', hairStyle: 'short', skin: '#e3b48e', smile: true }, prop: 'paper' }
    }
  };

  // what each court figure holds
  function courtProp(c, prop, o) {
    c.save(); ink(c); c.lineWidth = 2.4;
    const sk = o.skin;
    switch (prop) {
      case 'pizza':   // a pie on a wooden peel
        c.strokeStyle = '#7a5530'; c.lineWidth = 7; c.beginPath(); c.moveTo(60, -12); c.lineTo(52, -62); c.stroke(); ink(c);
        c.fillStyle = '#c9a27a'; c.beginPath(); c.ellipse(46, -92, 34, 12, -0.15, 0, TAU); c.fill(); c.stroke();
        c.fillStyle = '#d9a35a'; c.beginPath(); c.ellipse(46, -98, 28, 10, -0.15, 0, TAU); c.fill(); c.stroke();
        c.fillStyle = '#c0392b'; c.beginPath(); c.ellipse(46, -99, 23, 7.5, -0.15, 0, TAU); c.fill();
        c.fillStyle = '#f6dc8a'; for (const [x, y] of [[36, -100], [52, -103], [58, -97], [42, -95]]) { c.beginPath(); c.ellipse(x, y, 6, 2.5, 0, 0, TAU); c.fill(); }
        c.fillStyle = '#8a1a14'; for (const [x, y] of [[40, -101], [55, -99], [47, -96]]) { c.beginPath(); c.ellipse(x, y, 3.4, 1.6, 0, 0, TAU); c.fill(); }
        hand(c, 58, -26, sk); break;
      case 'cat':     // the bodega cat, sitting on his arm
        c.save(); c.translate(-52, -40);
        c.fillStyle = '#e08a3a'; c.beginPath(); c.ellipse(0, 6, 18, 16, 0, 0, TAU); c.fill(); c.stroke();
        c.beginPath(); c.arc(0, -16, 13, 0, TAU); c.fill(); c.stroke();
        for (const sx of [-1, 1]) { c.beginPath(); c.moveTo(sx * 4, -26); c.lineTo(sx * 12, -36); c.lineTo(sx * 13, -20); c.closePath(); c.fill(); c.stroke(); }
        c.strokeStyle = '#a85a1a'; c.lineWidth = 2; for (const y of [-2, 6, 14]) { c.beginPath(); c.moveTo(-14, y); c.quadraticCurveTo(-6, y - 4, 0, y); c.stroke(); }
        c.fillStyle = '#3a6a2a'; for (const sx of [-1, 1]) { c.beginPath(); c.ellipse(sx * 5, -17, 2.2, 3, 0, 0, TAU); c.fill(); }
        c.fillStyle = '#e8a0a0'; c.beginPath(); c.moveTo(-2, -12); c.lineTo(2, -12); c.lineTo(0, -10); c.closePath(); c.fill();
        c.strokeStyle = INK; c.lineWidth = 1; for (const sx of [-1, 1]) { c.beginPath(); c.moveTo(sx * 4, -11); c.lineTo(sx * 16, -13); c.moveTo(sx * 4, -10); c.lineTo(sx * 16, -8); c.stroke(); }
        c.strokeStyle = INK; c.lineWidth = 2.4; c.fillStyle = '#e08a3a'; c.beginPath(); c.moveTo(16, 14); c.quadraticCurveTo(30, 10, 26, -8); c.lineTo(22, -6); c.quadraticCurveTo(24, 6, 14, 8); c.closePath(); c.fill(); c.stroke();
        c.restore(); hand(c, -36, -22, sk); break;
      case 'sax':     // a brass saxophone
        c.strokeStyle = '#7a5a1e'; c.lineWidth = 15; c.lineCap = 'round';
        c.beginPath(); c.moveTo(34, -112); c.quadraticCurveTo(44, -100, 46, -86); c.lineTo(52, -32); c.quadraticCurveTo(56, -10, 70, -18); c.lineTo(76, -36); c.stroke();
        c.strokeStyle = '#d9b45a'; c.lineWidth = 11; c.stroke();
        c.strokeStyle = 'rgba(255,240,190,.6)'; c.lineWidth = 3; c.beginPath(); c.moveTo(45, -90); c.lineTo(50, -40); c.stroke();
        ink(c); c.fillStyle = '#e8c35a'; c.beginPath(); c.ellipse(77, -40, 12, 6, -0.3, 0, TAU); c.fill(); c.stroke();
        c.fillStyle = '#2a2018'; c.beginPath(); c.ellipse(77, -41, 7, 3, -0.3, 0, TAU); c.fill();
        c.fillStyle = '#fbf3d8'; for (const y of [-80, -68, -56, -44]) { c.beginPath(); c.arc(49 + (y + 80) * 0.08, y, 3, 0, TAU); c.fill(); c.lineWidth = 1.2; c.stroke(); }
        c.fillStyle = '#1a1a1a'; c.fillRect(30, -118, 6, 8);
        hand(c, 46, -70, sk); hand(c, 54, -40, sk); break;
      case 'keys':    // the super's ring of keys
        c.strokeStyle = '#a8b0b8'; c.lineWidth = 3.4; c.beginPath(); c.arc(-50, -50, 14, 0, TAU); c.stroke();
        c.fillStyle = '#d9b45a'; c.strokeStyle = INK; c.lineWidth = 1.6;
        [[-62, -34, 0.5], [-52, -32, 0.1], [-42, -34, -0.3], [-36, -40, -0.7]].forEach(([x, y, r], i) => {
          c.save(); c.translate(x, y); c.rotate(r); c.fillStyle = i % 2 ? '#a8b0b8' : '#d9b45a';
          c.beginPath(); c.arc(0, 0, 5, 0, TAU); c.fill(); c.stroke(); c.fillRect(-2, 4, 4, 22); c.strokeRect(-2, 4, 4, 22); c.fillRect(2, 18, 4, 3); c.fillRect(2, 23, 3, 3); c.restore();
        });
        hand(c, -50, -64, sk); break;
      case 'mic':     // an old-time microphone on a stand
        c.strokeStyle = '#3a3a40'; c.lineWidth = 4; c.beginPath(); c.moveTo(54, 0); c.lineTo(54, -78); c.stroke();
        c.fillStyle = '#c4ccd4'; c.strokeStyle = INK; c.lineWidth = 2.2; rrect(c, 42, -112, 24, 36, 11); c.fill(); c.stroke();
        c.strokeStyle = '#7a828a'; c.lineWidth = 1.2; for (let y = -106; y < -80; y += 5) { c.beginPath(); c.moveTo(45, y); c.lineTo(63, y); c.stroke(); }
        c.fillStyle = '#3a3a40'; c.fillRect(46, -78, 16, 5);
        hand(c, 54, -60, sk); break;
      case 'bouquet':
        c.fillStyle = '#e9dcc0'; c.beginPath(); c.moveTo(34, -6); c.lineTo(20, -52); c.lineTo(70, -52); c.lineTo(52, -6); c.closePath(); c.fill(); c.stroke();
        for (const [x, y, col] of [[28, -58, '#e8b4b8'], [44, -64, '#b23a2a'], [60, -58, '#f0d36a'], [36, -70, '#f6efe2'], [52, -72, '#9a7ab8'], [46, -54, '#e07a5f']]) { c.fillStyle = col; c.beginPath(); c.arc(x, y, 8, 0, TAU); c.fill(); c.stroke(); c.fillStyle = 'rgba(0,0,0,.18)'; c.beginPath(); c.arc(x, y, 2.5, 0, TAU); c.fill(); }
        hand(c, 42, -16, sk); break;
      case 'record':  // a vinyl record and her headphones
        c.save(); c.translate(-46, -52);
        c.fillStyle = '#141414'; c.beginPath(); c.arc(0, 0, 32, 0, TAU); c.fill(); c.stroke();
        c.strokeStyle = 'rgba(255,255,255,.14)'; c.lineWidth = 1; for (let r = 14; r < 30; r += 3.5) { c.beginPath(); c.arc(0, 0, r, 0, TAU); c.stroke(); }
        c.fillStyle = '#b23a2a'; c.beginPath(); c.arc(0, 0, 10, 0, TAU); c.fill(); c.fillStyle = '#f6efe0'; c.beginPath(); c.arc(0, 0, 2, 0, TAU); c.fill();
        c.restore();
        c.strokeStyle = '#2a2a30'; c.lineWidth = 5; c.beginPath(); c.arc(0, -86, 24, Math.PI * 1.15, Math.PI * 1.85, true); c.stroke();
        ink(c); c.fillStyle = '#e8c35a'; for (const sx of [-1, 1]) { rrect(c, sx * 22 - 6, -84, 12, 18, 5); c.fill(); c.stroke(); }
        hand(c, -30, -30, sk); break;
      case 'spray':   // a spray can and a puff of paint
        for (const [x, y, r, col] of [[70, -104, 9, 'rgba(232,176,74,.55)'], [80, -96, 7, 'rgba(178,58,42,.45)'], [76, -112, 6, 'rgba(58,122,154,.5)'], [86, -106, 5, 'rgba(232,176,74,.4)']]) { c.fillStyle = col; c.beginPath(); c.arc(x, y, r, 0, TAU); c.fill(); }
        c.save(); c.translate(56, -60); c.rotate(0.35);
        c.fillStyle = '#3a7a9a'; rrect(c, -10, -26, 20, 50, 4); c.fill(); c.stroke();
        c.fillStyle = '#f6efe0'; c.fillRect(-10, -8, 20, 12); c.strokeRect(-10, -8, 20, 12);
        c.fillStyle = '#c4ccd4'; rrect(c, -7, -36, 14, 10, 3); c.fill(); c.stroke(); c.fillStyle = '#1a1a1a'; c.fillRect(-3, -40, 6, 5);
        c.restore(); hand(c, 52, -44, sk); break;
      case 'boxes':   // two pizza boxes up on one hand
        c.fillStyle = '#e9d9b0';
        for (const y of [-48, -64]) { c.beginPath(); c.moveTo(-66, y); c.lineTo(-8, y); c.lineTo(-10, y + 14); c.lineTo(-64, y + 14); c.closePath(); c.fill(); c.stroke(); }
        c.fillStyle = '#b23a2a'; c.font = `700 9px ${DISP}`; c.textAlign = 'center'; c.fillText('HOT PIZZA', -37, -53); c.fillText('HOT PIZZA', -37, -37);
        hand(c, -60, -30, sk); hand(c, -12, -30, sk); break;
      case 'bagels':  // bagels stacked on a dowel
        c.strokeStyle = '#7a5530'; c.lineWidth = 4; c.beginPath(); c.moveTo(50, -6); c.lineTo(58, -120); c.stroke(); ink(c);
        for (let i = 0; i < 6; i++) { const y = -36 - i * 14, x = 52.5 + i * 0.95; c.fillStyle = i % 2 ? '#c98a3a' : '#d9a35a'; c.beginPath(); c.ellipse(x, y, 16, 7, 0, 0, TAU); c.fill(); c.stroke(); c.fillStyle = 'rgba(255,250,235,.8)'; for (const dx of [-8, -2, 5, 10]) { c.beginPath(); c.arc(x + dx, y - 2, 1, 0, TAU); c.fill(); } }
        hand(c, 50, -18, sk); break;
      case 'bat':     // a wooden bat over the shoulder
        c.save(); c.translate(54, -26); c.rotate(-0.55);
        c.fillStyle = '#c9a26a'; c.beginPath(); c.moveTo(-3, 0); c.lineTo(3, 0); c.lineTo(9, -120); c.quadraticCurveTo(0, -132, -9, -120); c.closePath(); c.fill(); c.stroke();
        c.fillStyle = '#1a1a22'; c.fillRect(-3.4, -14, 6.8, 14); c.beginPath(); c.ellipse(0, 2, 6, 3, 0, 0, TAU); c.fill(); c.stroke();
        c.restore(); hand(c, 54, -26, sk); break;
      case 'paper':
        c.save(); c.translate(-46, -40); c.rotate(-0.12);
        c.fillStyle = '#f2ebde'; c.fillRect(-20, -32, 40, 56); c.strokeRect(-20, -32, 40, 56);
        c.fillStyle = INK; c.font = `700 9px ${DISP}`; c.textAlign = 'center'; c.fillText('THE', 0, -21); c.fillText('DAILY', 0, -12);
        c.fillStyle = 'rgba(40,30,20,.35)'; for (let y = -4; y < 20; y += 5) c.fillRect(-15, y, 30, 2);
        c.restore(); hand(c, -32, -20, sk); break;
    }
    c.restore();
  }

  // a skyline silhouette: brownstones, towers and water tanks
  function skyline(c, x0, base, w, h, color, windows, seed = 11) {
    const R = rng(seed); c.save();
    let x = x0;
    while (x < x0 + w) {
      const bw = w * (0.04 + R() * 0.07), bh = h * (0.3 + R() * 0.7);
      c.fillStyle = color; c.fillRect(x, base - bh, bw + 1, bh);
      if (R() < 0.35) { // a water tower on the roof
        const tx = x + bw * (0.25 + R() * 0.4), tw = Math.min(bw * 0.4, h * 0.14), th = tw * 1.1;
        c.fillRect(tx, base - bh - th * 0.5, 2, th * 0.5); c.fillRect(tx + tw - 2, base - bh - th * 0.5, 2, th * 0.5);
        c.fillRect(tx, base - bh - th * 1.4, tw, th * 0.9);
        c.beginPath(); c.moveTo(tx - 1, base - bh - th * 1.4); c.lineTo(tx + tw / 2, base - bh - th * 1.85); c.lineTo(tx + tw + 1, base - bh - th * 1.4); c.fill();
      }
      if (windows) {
        for (let wy = base - bh + 6; wy < base - 6; wy += 9) for (let wx = x + 4; wx < x + bw - 4; wx += 7) if (R() < 0.28) { c.fillStyle = windows; c.fillRect(wx, wy, 3, 4); }
        c.fillStyle = color;
      }
      x += bw;
    }
    c.restore();
  }

  // brick wall behind a court figure
  function wallpaper(c, x, y, w, h, s) {
    const p = PAL[s];
    c.fillStyle = p.wall; c.fillRect(x, y, w, h);
    c.strokeStyle = 'rgba(120,70,50,.18)'; c.lineWidth = 1.2;
    for (let yy = y, row = 0; yy < y + h; yy += 14, row++) {
      c.beginPath(); c.moveTo(x, yy); c.lineTo(x + w, yy); c.stroke();
      for (let xx = x + (row % 2) * 16; xx < x + w; xx += 32) { c.beginPath(); c.moveTo(xx, yy); c.lineTo(xx, yy + 14); c.stroke(); }
    }
  }

  function courtCard(c, rank, s) {
    const color = PAL[s].ink, w = CW, h = CH;
    const fx = 76, fy = 64, fw = w - 152, fh = h - 128;
    const who = COURT[rank][s];
    c.save(); rrect(c, fx, fy, fw, fh, 8); c.clip();
    wallpaper(c, fx, fy, fw, fh, s);
    const glow = c.createRadialGradient(fx + fw / 2, fy + fh * 0.42, 10, fx + fw / 2, fy + fh * 0.42, fh * 0.6);
    glow.addColorStop(0, 'rgba(255,248,232,.85)'); glow.addColorStop(1, 'rgba(255,248,225,0)'); c.fillStyle = glow; c.fillRect(fx, fy, fw, fh);
    // a window onto the skyline behind the head
    c.save(); c.globalAlpha = 0.5;
    const wx = fx + 26, wy = fy + 34, ww = fw - 52, wh = 120;
    const sky = c.createLinearGradient(0, wy, 0, wy + wh); sky.addColorStop(0, '#5a6a9a'); sky.addColorStop(1, '#f2b880'); c.fillStyle = sky; c.fillRect(wx, wy, ww, wh);
    skyline(c, wx, wy + wh, ww, 60, '#3a3050', null, rank.charCodeAt(0) + s.charCodeAt(0));
    c.strokeStyle = GOLD; c.lineWidth = 3; c.strokeRect(wx, wy, ww, wh); c.beginPath(); c.moveTo(wx + ww / 2, wy); c.lineTo(wx + ww / 2, wy + wh); c.moveTo(wx, wy + wh / 2); c.lineTo(wx + ww, wy + wh / 2); c.lineWidth = 2; c.stroke();
    c.restore();
    const fit = 1.48, o = who.o;
    c.save(); c.translate(fx + fw / 2, fy + fh + 2); c.scale(fit, fit);
    bust(c, o); courtProp(c, who.prop, o);
    c.restore();
    // the name plate
    c.fillStyle = 'rgba(255,250,238,.92)'; rrect(c, fx + 16, fy + fh - 30, fw - 32, 22, 4); c.fill();
    c.strokeStyle = GOLD; c.lineWidth = 1.2; c.stroke();
    c.fillStyle = color; c.font = `600 13px ${DISP}`; c.textAlign = 'center'; c.fillText(who.name.toUpperCase(), fx + fw / 2, fy + fh - 14);
    c.restore();
    c.strokeStyle = GOLD; c.lineWidth = 3; rrect(c, fx, fy, fw, fh, 8); c.stroke();
    c.lineWidth = 1; rrect(c, fx + 5, fy + 5, fw - 10, fh - 10, 5); c.stroke();
    suit(c, s, fx + 22, fy + 22, 24, color, false);
    suit(c, s, fx + fw - 22, fy + 22, 24, color, false);
  }

  const PIPS = {
    2: [[.5, 0], [.5, 1]], 3: [[.5, 0], [.5, .5], [.5, 1]], 4: [[0, 0], [1, 0], [0, 1], [1, 1]],
    5: [[0, 0], [1, 0], [.5, .5], [0, 1], [1, 1]], 6: [[0, 0], [1, 0], [0, .5], [1, .5], [0, 1], [1, 1]],
    7: [[0, 0], [1, 0], [.5, .25], [0, .5], [1, .5], [0, 1], [1, 1]],
    8: [[0, 0], [1, 0], [.5, .25], [0, .5], [1, .5], [.5, .75], [0, 1], [1, 1]],
    9: [[0, 0], [1, 0], [0, 1 / 3], [1, 1 / 3], [.5, .5], [0, 2 / 3], [1, 2 / 3], [0, 1], [1, 1]],
    10: [[0, 0], [1, 0], [.5, 1 / 6], [0, 1 / 3], [1, 1 / 3], [0, 2 / 3], [1, 2 / 3], [.5, 5 / 6], [0, 1], [1, 1]]
  };
  function pipCard(c, n, s) {
    const color = PAL[s].ink, x0 = 114, x1 = CW - 114, y0 = 104, y1 = CH - 104;
    for (const [px, py] of PIPS[n]) {
      const x = x0 + (x1 - x0) * px, y = y0 + (y1 - y0) * py;
      if (py > 0.5) { c.save(); c.translate(x, y); c.rotate(Math.PI); suit(c, s, 0, 0, 56, color); c.restore(); }
      else suit(c, s, x, y, 56, color);
    }
  }
  function aceCard(c, s) {
    const color = PAL[s].ink, cx = CW / 2, cy = CH / 2;
    c.save();
    // an Art Deco sunburst
    c.strokeStyle = 'rgba(184,145,63,.35)'; c.lineWidth = 2;
    for (let i = 0; i < 36; i++) { const a = i / 36 * TAU; c.beginPath(); c.moveTo(cx + Math.cos(a) * 70, cy + Math.sin(a) * 70); c.lineTo(cx + Math.cos(a) * 128, cy + Math.sin(a) * 158); c.stroke(); }
    c.strokeStyle = GOLD; c.lineWidth = 2; c.beginPath(); c.ellipse(cx, cy, 104, 132, 0, 0, TAU); c.stroke();
    c.lineWidth = 1; c.beginPath(); c.ellipse(cx, cy, 110, 138, 0, 0, TAU); c.stroke();
    c.restore();
    if (s === 'S') { // the Ace of Spades: the boss card, with the skyline inside
      c.save(); c.fillStyle = BLACK; c.beginPath(); suitPath(c, 'S', cx, cy, 170); c.fill(); c.clip();
      const g = c.createLinearGradient(0, cy - 90, 0, cy + 60); g.addColorStop(0, '#1f2a4a'); g.addColorStop(0.7, '#5a3a5a'); g.addColorStop(1, '#d9805a');
      c.fillStyle = g; c.beginPath(); suitPath(c, 'S', cx, cy, 150); c.fill();
      c.fillStyle = 'rgba(255,250,230,.8)'; const R = rng(5); for (let i = 0; i < 26; i++) { c.beginPath(); c.arc(cx - 70 + R() * 140, cy - 80 + R() * 70, R() * 1.4 + 0.4, 0, TAU); c.fill(); }
      skyline(c, cx - 90, cy + 50, 180, 70, '#141018', '#ffd27a', 23);
      c.restore();
      c.fillStyle = GOLD2; c.strokeStyle = '#6a4a1e'; c.lineWidth = 1.5;
      c.beginPath(); c.moveTo(cx - 92, cy + 70); c.lineTo(cx + 92, cy + 70); c.lineTo(cx + 84, cy + 96); c.lineTo(cx - 84, cy + 96); c.closePath(); c.fill(); c.stroke();
      c.fillStyle = '#18141a'; c.font = `700 17px ${DISP}`; c.textAlign = 'center'; c.fillText('B R O O K L Y N', cx, cy + 90);
      return;
    }
    suit(c, s, cx, cy, 132, color);
  }

  /* ---------- the card back: night navy and gold, with the bridge ---------- */
  function bridge(c, x, base, h, color) {   // a stone tower with two pointed arches, and its cables
    const w = h * 0.36;
    c.save(); c.translate(x, base); c.fillStyle = color; c.strokeStyle = color;
    c.lineWidth = Math.max(1, h * 0.012);
    for (const sx of [-1, 1]) { // the cables sweeping down to each side
      c.beginPath(); c.moveTo(sx * w * 0.45, -h * 0.92); c.quadraticCurveTo(sx * w * 1.1, -h * 0.3, sx * w * 2.6, -h * 0.28); c.stroke();
      c.globalAlpha = 0.5; for (let i = 1; i < 8; i++) { const t = i / 8, px = sx * (w * 0.45 + (w * 2.15) * t); c.beginPath(); c.moveTo(sx * w * 0.45, -h * 0.92); c.lineTo(px, -h * 0.28); c.stroke(); } c.globalAlpha = 1;
    }
    c.fillRect(-w * 2.6, -h * 0.3, w * 5.2, h * 0.05); // the deck
    c.beginPath(); c.rect(-w / 2, -h, w, h);
    for (const sx of [-1, 1]) { const ax = sx * w * 0.2; c.moveTo(ax - w * 0.13, -h * 0.34); c.lineTo(ax - w * 0.13, -h * 0.7); c.lineTo(ax, -h * 0.8); c.lineTo(ax + w * 0.13, -h * 0.7); c.lineTo(ax + w * 0.13, -h * 0.34); c.closePath(); }
    c.fill('evenodd');
    c.fillRect(-w * 0.6, -h * 1.02, w * 1.2, h * 0.04); c.fillRect(-w * 0.6, -h * 0.36, w * 1.2, h * 0.03);
    c.restore();
  }
  function backCard() {
    const cv = canvas(CW, CH), c = cv.getContext('2d');
    rrect(c, 1, 1, CW - 2, CH - 2, 22); c.fillStyle = '#f6eedb'; c.fill();
    rrect(c, 12, 12, CW - 24, CH - 24, 14); const g = c.createLinearGradient(0, 0, 0, CH); g.addColorStop(0, '#1f2a4a'); g.addColorStop(1, '#0e1428'); c.fillStyle = g; c.fill();
    c.save(); c.clip();
    // a Deco lattice of little gold spades
    c.strokeStyle = 'rgba(232,195,90,.22)'; c.lineWidth = 1.2;
    for (let k = -CH; k < CW + CH; k += 30) { c.beginPath(); c.moveTo(k, 0); c.lineTo(k + CH, CH); c.moveTo(k, CH); c.lineTo(k + CH, 0); c.stroke(); }
    c.fillStyle = 'rgba(232,195,90,.35)';
    for (let y = 12; y < CH; y += 30) for (let x = 12 + ((y - 12) / 30 % 2) * 15; x < CW; x += 30) { c.beginPath(); suitPath(c, 'S', x, y + 15, 10); c.fill(); }
    c.restore();
    c.strokeStyle = '#e8c35a'; c.lineWidth = 3; rrect(c, 26, 26, CW - 52, CH - 52, 8); c.stroke();
    c.lineWidth = 1; rrect(c, 32, 32, CW - 64, CH - 64, 5); c.stroke();
    // the medallion: a dusk sky with the bridge tower
    const cx = CW / 2, cy = CH / 2;
    c.save(); c.beginPath(); c.ellipse(cx, cy, 94, 124, 0, 0, TAU);
    const sg = c.createLinearGradient(0, cy - 124, 0, cy + 124); sg.addColorStop(0, '#141a36'); sg.addColorStop(0.6, '#4a2a52'); sg.addColorStop(1, '#c8664a'); c.fillStyle = sg; c.fill(); c.clip();
    skyline(c, cx - 100, cy + 124, 200, 70, 'rgba(14,10,20,.85)', 'rgba(255,210,122,.8)', 3);
    bridge(c, cx, cy + 80, 170, '#e8c35a');
    c.restore();
    c.strokeStyle = '#e8c35a'; c.lineWidth = 3; c.beginPath(); c.ellipse(cx, cy, 94, 124, 0, 0, TAU); c.stroke();
    c.lineWidth = 1; c.beginPath(); c.ellipse(cx, cy, 86, 116, 0, 0, TAU); c.stroke();
    c.fillStyle = '#f2d68e'; c.font = `26px ${SCRIPT}`; c.textAlign = 'center'; c.fillText('Spades', cx, CH - 50);
    c.font = `600 15px ${DISP}`; c.fillText('B R O O K L Y N', cx, 66);
    rrect(c, 1, 1, CW - 2, CH - 2, 22); c.strokeStyle = 'rgba(70,50,30,.45)'; c.lineWidth = 2; c.stroke();
    return cv;
  }

  /* ---------- public: a card face ---------- */
  const RANK = { 14: 'A', 11: 'J', 12: 'Q', 13: 'K' };
  function face(card) {
    const cv = canvas(CW, CH), c = cv.getContext('2d');
    paper(c, CW, CH, card.r * 7 + 'SHDC'.indexOf(card.s) * 131);
    const color = PAL[card.s].ink;
    border(c, CW, CH, card.s === 'H' || card.s === 'D' ? 'rgba(160,28,48,.75)' : GOLD);
    const label = RANK[card.r] || String(card.r);
    index(c, label, card.s, color, CW, CH);
    if (card.r === 14) aceCard(c, card.s);
    else if (card.r > 10) courtCard(c, label, card.s);
    else pipCard(c, card.r, card.s);
    return cv;
  }

  /* ---------- rooftop props, seen from above ---------- */
  function soda(size) {   // an ice-cold soda on a cork coaster
    const cv = canvas(size, size), c = cv.getContext('2d'), k = size / 220; c.scale(k, k);
    c.fillStyle = 'rgba(0,0,0,.35)'; c.beginPath(); c.ellipse(116, 122, 90, 86, 0, 0, TAU); c.fill();
    c.strokeStyle = INK; c.lineWidth = 3;
    let g = c.createRadialGradient(100, 96, 20, 110, 110, 90); g.addColorStop(0, '#d9b07a'); g.addColorStop(1, '#a8804a');
    c.fillStyle = g; c.beginPath(); c.arc(110, 110, 86, 0, TAU); c.fill(); c.stroke();
    const R = rng(9); c.fillStyle = 'rgba(90,60,30,.35)'; for (let i = 0; i < 160; i++) { const a = R() * TAU, r = R() * 82; c.fillRect(110 + Math.cos(a) * r, 110 + Math.sin(a) * r, 2, 2); }
    // the glass
    c.fillStyle = 'rgba(220,235,240,.55)'; c.beginPath(); c.arc(106, 106, 62, 0, TAU); c.fill(); c.stroke();
    g = c.createRadialGradient(98, 96, 8, 106, 106, 54); g.addColorStop(0, '#d9783a'); g.addColorStop(0.8, '#a8461e'); g.addColorStop(1, '#7a2e10');
    c.fillStyle = g; c.beginPath(); c.arc(106, 106, 54, 0, TAU); c.fill();
    // ice
    for (const [x, y, r] of [[86, 92, 0.3], [122, 98, -0.4], [104, 128, 0.9]]) { c.save(); c.translate(x, y); c.rotate(r); c.fillStyle = 'rgba(240,250,255,.75)'; c.strokeStyle = 'rgba(255,255,255,.95)'; c.lineWidth = 2; rrect(c, -15, -15, 30, 30, 6); c.fill(); c.stroke(); c.restore(); }
    // a lemon wheel and a striped straw
    c.fillStyle = '#f6dc5a'; c.strokeStyle = '#c8a82a'; c.lineWidth = 2; c.beginPath(); c.arc(134, 128, 18, 0, TAU); c.fill(); c.stroke();
    c.strokeStyle = 'rgba(200,170,40,.7)'; c.lineWidth = 1.2; for (let i = 0; i < 8; i++) { const a = i / 8 * TAU; c.beginPath(); c.moveTo(134, 128); c.lineTo(134 + Math.cos(a) * 15, 128 + Math.sin(a) * 15); c.stroke(); }
    c.save(); c.translate(112, 100); c.rotate(-0.7); c.fillStyle = '#fbf8f0'; c.strokeStyle = INK; c.lineWidth = 2; rrect(c, -5, -80, 10, 84, 5); c.fill(); c.stroke();
    c.fillStyle = '#b23a2a'; for (let y = -76; y < 0; y += 12) c.fillRect(-4, y, 8, 5); c.restore();
    c.fillStyle = 'rgba(255,255,255,.4)'; c.beginPath(); c.ellipse(80, 72, 16, 6, -0.7, 0, TAU); c.fill();
    return cv;
  }
  function pizza(size) {  // a slice on a paper plate
    const cv = canvas(size, size), c = cv.getContext('2d'), k = size / 220; c.scale(k, k);
    c.fillStyle = 'rgba(0,0,0,.35)'; c.beginPath(); c.ellipse(114, 120, 96, 90, 0, 0, TAU); c.fill();
    c.strokeStyle = INK; c.lineWidth = 3;
    let g = c.createRadialGradient(100, 96, 20, 110, 110, 94); g.addColorStop(0, '#fffdf8'); g.addColorStop(1, '#e6ded0');
    c.fillStyle = g; c.beginPath(); c.arc(110, 110, 92, 0, TAU); c.fill(); c.stroke();
    c.strokeStyle = 'rgba(0,0,0,.1)'; c.lineWidth = 1.2; for (let i = 0; i < 48; i++) { const a = i / 48 * TAU; c.beginPath(); c.moveTo(110 + Math.cos(a) * 74, 110 + Math.sin(a) * 74); c.lineTo(110 + Math.cos(a) * 90, 110 + Math.sin(a) * 90); c.stroke(); }
    c.save(); c.translate(110, 112); c.rotate(-0.4);
    c.fillStyle = '#e8b04a'; c.strokeStyle = '#7a4a14'; c.lineWidth = 2.6;
    c.beginPath(); c.moveTo(0, 70); c.lineTo(-56, -42); c.quadraticCurveTo(0, -66, 56, -42); c.closePath(); c.fill(); c.stroke();
    c.fillStyle = '#c98a3a'; c.beginPath(); c.moveTo(-56, -42); c.quadraticCurveTo(0, -66, 56, -42); c.lineTo(50, -30); c.quadraticCurveTo(0, -52, -50, -30); c.closePath(); c.fill(); c.stroke();
    c.fillStyle = '#f6d56a'; c.beginPath(); c.moveTo(0, 58); c.lineTo(-44, -26); c.quadraticCurveTo(0, -44, 44, -26); c.closePath(); c.fill();
    c.fillStyle = 'rgba(200,60,30,.35)'; for (const [x, y, r] of [[-18, -10, 9], [14, 4, 8], [-4, 26, 6], [22, -18, 7]]) { c.beginPath(); c.arc(x, y, r, 0, TAU); c.fill(); }
    c.fillStyle = '#a8261a'; c.strokeStyle = '#6a140a'; c.lineWidth = 1.6; for (const [x, y] of [[-16, -16], [16, -12], [0, 10], [-6, 34], [24, -26]]) { c.beginPath(); c.arc(x, y, 7.5, 0, TAU); c.fill(); c.stroke(); }
    c.fillStyle = '#3a6a2a'; for (const [x, y] of [[-24, 4], [8, 22], [30, -8]]) { c.beginPath(); c.ellipse(x, y, 3, 1.4, 0.6, 0, TAU); c.fill(); }
    c.restore();
    return cv;
  }

  /* ---------- the tabletop: weathered deck boards ---------- */
  function deck(W, H) {
    const cv = canvas(W, H), c = cv.getContext('2d'), img = c.createImageData(W, H), d = img.data;
    const P = new Uint8Array(512); { const R = rng(1898); const p = [...Array(256).keys()]; for (let i = 255; i > 0; i--) { const j = Math.floor(R() * (i + 1)); [p[i], p[j]] = [p[j], p[i]]; } for (let i = 0; i < 512; i++) P[i] = p[i & 255]; }
    const fade = t => t * t * t * (t * (t * 6 - 15) + 10), lerp = (a, b, t) => a + (b - a) * t;
    const grad = (h, x, y) => { const v = h & 3; return (v & 1 ? -x : x) + (v & 2 ? -y : y); };
    function noise(x, y) { const X = Math.floor(x) & 255, Y = Math.floor(y) & 255; x -= Math.floor(x); y -= Math.floor(y); const u = fade(x), v = fade(y); const a = P[X] + Y, b = P[X + 1] + Y; return lerp(lerp(grad(P[a], x, y), grad(P[b], x - 1, y), u), lerp(grad(P[a + 1], x, y - 1), grad(P[b + 1], x - 1, y - 1), u), v); }
    function fbm(x, y, o) { let s = 0, a = 1, f = 1; for (let i = 0; i < o; i++) { s += a * noise(x * f, y * f); a *= 0.5; f *= 2.03; } return s; }
    const plank = 104, R = rng(77), tone = [], joint = [];
    for (let i = 0; i < Math.ceil(H / plank) + 1; i++) { tone.push(0.85 + R() * 0.3); joint.push(R() * W); }
    for (let y = 0; y < H; y++) {
      const pi = Math.floor(y / plank), py = y - pi * plank;
      for (let x = 0; x < W; x++) {
        const t = tone[pi];
        const grain = fbm(x / 260 + pi * 13, y / 9, 4);
        const ring = Math.abs(Math.sin((y / 7 + grain * 6 + pi * 3) ));
        let v = 0.55 + 0.25 * grain + 0.12 * Math.pow(ring, 6) + noise(x / 2.5, y / 2.5) * 0.05;
        let r = 118 * v * t, g = 86 * v * t, b = 60 * v * t;
        const gry = 0.25 + 0.2 * fbm(x / 300 + 5, y / 300, 3); // sun-bleached grey
        r = lerp(r, 120 * v, gry); g = lerp(g, 108 * v, gry); b = lerp(b, 96 * v, gry);
        if (py < 3 || py > plank - 2) { r *= 0.35; g *= 0.32; b *= 0.3; }                   // the gaps
        if (Math.abs(x - joint[pi]) < 1.5) { r *= 0.45; g *= 0.42; b *= 0.4; }              // a board end
        for (const nx of [joint[pi] - 14, joint[pi] + 14]) for (const ny of [16, plank - 16]) if (Math.hypot(x - nx, py - ny) < 3) { r = 60; g = 56; b = 54; }
        const i = (y * W + x) * 4; d[i] = r; d[i + 1] = g; d[i + 2] = b; d[i + 3] = 255;
      }
    }
    c.putImageData(img, 0, 0);
    return cv;
  }

  /* ---------- string lights strung across the top of the table ---------- */
  function lights(W, H) {
    const cv = canvas(W, H), c = cv.getContext('2d');
    const swags = 4, sag = H * 0.42, top = H * 0.08;
    const cy = x => { const t = (x / W * swags) % 1; return top + sag * 4 * t * (1 - t); };
    c.strokeStyle = 'rgba(16,10,8,.9)'; c.lineWidth = 3; c.beginPath(); for (let x = 0; x <= W; x += 4) { x ? c.lineTo(x, cy(x)) : c.moveTo(x, cy(x)); } c.stroke();
    const n = swags * 7;
    for (let i = 0; i < n; i++) {
      const x = W * (i + 0.5) / n, y = cy(x) + 4;
      const gl = c.createRadialGradient(x, y + 14, 2, x, y + 14, 48); gl.addColorStop(0, 'rgba(255,214,130,.55)'); gl.addColorStop(1, 'rgba(255,190,110,0)');
      c.fillStyle = gl; c.beginPath(); c.arc(x, y + 14, 48, 0, TAU); c.fill();
      c.fillStyle = '#1a120c'; c.fillRect(x - 4, y - 2, 8, 9);
      const bg = c.createRadialGradient(x - 2, y + 12, 1, x, y + 14, 9); bg.addColorStop(0, '#fffbe8'); bg.addColorStop(1, '#ffc860');
      c.fillStyle = bg; c.beginPath(); c.ellipse(x, y + 15, 7, 9, 0, 0, TAU); c.fill();
    }
    return cv;
  }

  /* ---------- the icon: the Ace of Spades over a dusk skyline ---------- */
  function icon(size) {
    const cv = canvas(size, size), c = cv.getContext('2d'), s = size / 512;
    c.scale(s, s);
    rrect(c, 16, 16, 480, 480, 96); const g = c.createLinearGradient(0, 16, 0, 496); g.addColorStop(0, '#1f2a4a'); g.addColorStop(0.65, '#4a2a52'); g.addColorStop(1, '#c8664a'); c.fillStyle = g; c.fill();
    c.save(); rrect(c, 16, 16, 480, 480, 96); c.clip();
    skyline(c, 16, 496, 480, 150, 'rgba(14,10,20,.9)', size >= 64 ? 'rgba(255,210,122,.85)' : null, 41);
    c.restore();
    c.strokeStyle = '#e8c35a'; c.lineWidth = 12; rrect(c, 40, 40, 432, 432, 76); c.stroke();
    c.save(); c.translate(256, 248); c.rotate(-0.1);
    rrect(c, -110, -150, 220, 300, 24); c.fillStyle = '#fffaf0'; c.fill(); c.strokeStyle = '#1e1916'; c.lineWidth = 7; c.stroke();
    suit(c, 'S', 0, 4, 170, '#18141a'); c.restore();
    return cv;
  }

  return { face, back: backCard, soda, pizza, deck, lights, skyline, bridge, icon, heartPath, suit, CW, CH, PAL, DISP, SERIF, SCRIPT, COURT };
})();
