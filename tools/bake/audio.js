'use strict';
/* =====================================================================
   SPADES - the rooftop music and sound effects, synthesized with WebAudio.
   Original compositions (c) Ironite:
     "Fire Escape Beat"  - a lo-fi boom-bap groove on electric piano
     "Stoop Soul"        - a soul strut for organ and tenor sax
     "Bridge Lights"     - a late-night jazz ballad for piano and brushes
   bake.html renders each with an OfflineAudioContext into the game's files.
   ===================================================================== */
const Sound = (() => {
  const mtof = m => 440 * Math.pow(2, (m - 69) / 12);
  let noiseBuf = null;
  function noise(ctx) {
    if (noiseBuf && noiseBuf.sampleRate === ctx.sampleRate) return noiseBuf;
    const n = ctx.sampleRate * 2, b = ctx.createBuffer(1, n, ctx.sampleRate), d = b.getChannelData(0);
    let s = 12345; for (let i = 0; i < n; i++) { s = (s * 1103515245 + 12345) & 0x7fffffff; d[i] = s / 0x3fffffff - 1; }
    return (noiseBuf = b);
  }
  function env(g, t, a, d, sus, r, peak, len) {   // ADSR on a gain param
    g.setValueAtTime(0, t); g.linearRampToValueAtTime(peak, t + a);
    g.linearRampToValueAtTime(peak * sus, t + a + d);
    g.setValueAtTime(peak * sus, Math.max(t + a + d, t + len));
    g.linearRampToValueAtTime(0, Math.max(t + a + d, t + len) + r);
  }
  function reverb(ctx, secs = 1.8, wet = 0.22) {
    const len = Math.floor(ctx.sampleRate * secs), ir = ctx.createBuffer(2, len, ctx.sampleRate);
    for (let ch = 0; ch < 2; ch++) { const d = ir.getChannelData(ch); let s = 99 + ch; for (let i = 0; i < len; i++) { s = (s * 1103515245 + 12345) & 0x7fffffff; d[i] = (s / 0x3fffffff - 1) * Math.pow(1 - i / len, 3); } }
    const conv = ctx.createConvolver(); conv.buffer = ir;
    const inp = ctx.createGain(), dry = ctx.createGain(), w = ctx.createGain();
    dry.gain.value = 1; w.gain.value = wet;
    inp.connect(dry); inp.connect(conv); conv.connect(w);
    const out = ctx.createGain(); dry.connect(out); w.connect(out);
    return { inp, out };
  }

  /* ---------------- instruments ---------------- */
  // a plucked string (guitar, double bass): a bright saw that darkens as it rings
  function pluck(ctx, dest, t, m, dur, vel, bright = 3200, decay = 0.6) {
    const o = ctx.createOscillator(); o.type = 'sawtooth'; o.frequency.value = mtof(m);
    const o2 = ctx.createOscillator(); o2.type = 'triangle'; o2.frequency.value = mtof(m);
    const lp = ctx.createBiquadFilter(); lp.type = 'lowpass'; lp.Q.value = 1;
    lp.frequency.setValueAtTime(bright, t); lp.frequency.exponentialRampToValueAtTime(Math.max(200, bright * 0.18), t + decay);
    const g = ctx.createGain(); g.gain.setValueAtTime(0, t); g.gain.linearRampToValueAtTime(vel * 0.22, t + 0.004);
    g.gain.exponentialRampToValueAtTime(0.0005, t + Math.max(dur, 0.05) + decay);
    const g2 = ctx.createGain(); g2.gain.value = 0.6; o2.connect(g2); g2.connect(lp);
    o.connect(lp); lp.connect(g); g.connect(dest);
    o.start(t); o.stop(t + dur + decay + 0.05); o2.start(t); o2.stop(t + dur + decay + 0.05);
  }
  function bass(ctx, dest, t, m, dur, vel) {
    const o = ctx.createOscillator(); o.type = 'triangle'; o.frequency.value = mtof(m);
    const s = ctx.createOscillator(); s.type = 'sine'; s.frequency.value = mtof(m);
    const g = ctx.createGain(); g.gain.setValueAtTime(0, t); g.gain.linearRampToValueAtTime(vel * 0.42, t + 0.01);
    g.gain.exponentialRampToValueAtTime(vel * 0.12, t + dur * 0.6); g.gain.linearRampToValueAtTime(0, t + dur + 0.05);
    const lp = ctx.createBiquadFilter(); lp.type = 'lowpass'; lp.frequency.value = 900;
    o.connect(lp); s.connect(lp); lp.connect(g); g.connect(dest);
    o.start(t); o.stop(t + dur + 0.1); s.start(t); s.stop(t + dur + 0.1);
  }
  // a piano: a few decaying partials
  function piano(ctx, dest, t, m, dur, vel) {
    const f = mtof(m), ring = Math.min(3.2, 0.6 + dur * 1.2) * (m < 60 ? 1.3 : 1);
    const out = ctx.createGain(); out.gain.value = vel * 0.2; out.connect(dest);
    for (const [k, a, det] of [[1, 0.62, 0], [1, 0.3, 4], [2, 0.22, 0], [3, 0.1, 0], [4, 0.05, 0], [5.02, 0.03, 0]]) {
      const o = ctx.createOscillator(); o.type = 'sine'; o.frequency.value = f * k; o.detune.value = det;
      const g = ctx.createGain(); g.gain.setValueAtTime(0, t); g.gain.linearRampToValueAtTime(a, t + 0.006);
      g.gain.exponentialRampToValueAtTime(0.0008, t + ring / Math.sqrt(k));
      o.connect(g); g.connect(out); o.start(t); o.stop(t + ring + 0.05);
    }
  }
  function pad(ctx, dest, t, m, dur, vel) {
    for (const det of [-7, 7]) {
      const o = ctx.createOscillator(); o.type = 'sawtooth'; o.frequency.value = mtof(m); o.detune.value = det;
      const lp = ctx.createBiquadFilter(); lp.type = 'lowpass'; lp.frequency.value = 900;
      const g = ctx.createGain(); env(g.gain, t, 0.5, 0.3, 0.8, 0.8, vel * 0.04, dur);
      o.connect(lp); lp.connect(g); g.connect(dest); o.start(t); o.stop(t + dur + 1);
    }
  }

  // an electric piano: a sine with a bell-like FM attack and a slow tremolo
  function rhodes(ctx, dest, t, m, dur, vel) {
    const f = mtof(m), ring = Math.min(3.5, dur + 1.4);
    const car = ctx.createOscillator(); car.type = 'sine'; car.frequency.value = f;
    const mod = ctx.createOscillator(); mod.type = 'sine'; mod.frequency.value = f;
    const mg = ctx.createGain(); mg.gain.setValueAtTime(f * 1.6 * vel, t); mg.gain.exponentialRampToValueAtTime(f * 0.08, t + 0.5);
    mod.connect(mg); mg.connect(car.frequency);
    const g = ctx.createGain(); g.gain.setValueAtTime(0, t); g.gain.linearRampToValueAtTime(vel * 0.22, t + 0.006);
    g.gain.exponentialRampToValueAtTime(vel * 0.08, t + 0.4); g.gain.setValueAtTime(vel * 0.08, t + Math.max(0.4, dur));
    g.gain.exponentialRampToValueAtTime(0.0005, t + Math.max(0.4, dur) + 0.5);
    const trem = ctx.createGain(); trem.gain.value = 1; const lfo = ctx.createOscillator(); lfo.frequency.value = 4.2; const lg = ctx.createGain(); lg.gain.value = 0.12; lfo.connect(lg); lg.connect(trem.gain);
    car.connect(g); g.connect(trem); trem.connect(dest);
    for (const o of [car, mod, lfo]) { o.start(t); o.stop(t + ring + 0.1); }
  }
  // a drawbar organ: a few sine partials through a wobbling speaker
  function organ(ctx, dest, t, m, dur, vel) {
    const f = mtof(m), out = ctx.createGain(); env(out.gain, t, 0.01, 0.05, 0.9, 0.08, vel * 0.09, dur);
    const trem = ctx.createGain(); trem.gain.value = 1; const lfo = ctx.createOscillator(); lfo.frequency.value = 6.4; const lg = ctx.createGain(); lg.gain.value = 0.18; lfo.connect(lg); lg.connect(trem.gain);
    for (const [k, a] of [[0.5, 0.6], [1, 1], [2, 0.7], [3, 0.35], [4, 0.2]]) {
      const o = ctx.createOscillator(); o.type = 'sine'; o.frequency.value = f * k;
      const og = ctx.createGain(); og.gain.value = a; o.connect(og); og.connect(out); o.start(t); o.stop(t + dur + 0.2);
    }
    out.connect(trem); trem.connect(dest); lfo.start(t); lfo.stop(t + dur + 0.2);
  }
  // a tenor sax: a reedy saw and square through a vocal bandpass, with vibrato that blooms in
  function sax(ctx, dest, t, m, dur, vel, bend = 0) {
    const f = mtof(m), out = ctx.createGain(); env(out.gain, t, 0.03, 0.08, 0.78, 0.09, vel * 0.13, dur);
    const bp = ctx.createBiquadFilter(); bp.type = 'bandpass'; bp.frequency.value = 1300; bp.Q.value = 0.9;
    const lp = ctx.createBiquadFilter(); lp.type = 'lowpass'; lp.frequency.setValueAtTime(900, t); lp.frequency.linearRampToValueAtTime(3200, t + 0.08); lp.frequency.linearRampToValueAtTime(2000, t + dur);
    const vib = ctx.createOscillator(); vib.frequency.value = 5.2; const vg = ctx.createGain(); vg.gain.setValueAtTime(0, t); vg.gain.linearRampToValueAtTime(14, t + Math.min(0.4, dur));
    vib.connect(vg);
    for (const [typ, g] of [['sawtooth', 0.7], ['square', 0.35]]) {
      const o = ctx.createOscillator(); o.type = typ; o.frequency.setValueAtTime(f, t); if (bend) o.frequency.linearRampToValueAtTime(f * Math.pow(2, bend / 12), t + dur);
      vg.connect(o.detune); const og = ctx.createGain(); og.gain.value = g; o.connect(og); og.connect(lp); o.start(t); o.stop(t + dur + 0.2);
    }
    const mix = ctx.createGain(); lp.connect(bp); lp.connect(mix); bp.connect(mix); mix.gain.value = 0.7; mix.connect(out); out.connect(dest);
    vib.start(t); vib.stop(t + dur + 0.2);
  }
  function upright(ctx, dest, t, m, dur, vel) {   // an upright bass: a round thump that rings briefly
    const o = ctx.createOscillator(); o.type = 'triangle'; o.frequency.value = mtof(m);
    const s = ctx.createOscillator(); s.type = 'sine'; s.frequency.value = mtof(m);
    const lp = ctx.createBiquadFilter(); lp.type = 'lowpass'; lp.frequency.setValueAtTime(1400, t); lp.frequency.exponentialRampToValueAtTime(300, t + 0.25);
    const g = ctx.createGain(); g.gain.setValueAtTime(0, t); g.gain.linearRampToValueAtTime(vel * 0.5, t + 0.008); g.gain.exponentialRampToValueAtTime(vel * 0.12, t + 0.3); g.gain.exponentialRampToValueAtTime(0.0005, t + dur + 0.2);
    o.connect(lp); s.connect(lp); lp.connect(g); g.connect(dest); o.start(t); o.stop(t + dur + 0.3); s.start(t); s.stop(t + dur + 0.3);
  }
  function kick(ctx, dest, t, vel = 1) {
    const o = ctx.createOscillator(); o.frequency.setValueAtTime(130, t); o.frequency.exponentialRampToValueAtTime(42, t + 0.14);
    const g = ctx.createGain(); g.gain.setValueAtTime(vel * 0.9, t); g.gain.exponentialRampToValueAtTime(0.001, t + 0.38);
    o.connect(g); g.connect(dest); o.start(t); o.stop(t + 0.4);
  }
  function snare(ctx, dest, t, vel = 1, lofi = false) {
    const s = ctx.createBufferSource(); s.buffer = noise(ctx);
    const bp = ctx.createBiquadFilter(); bp.type = 'bandpass'; bp.frequency.value = lofi ? 1500 : 2200; bp.Q.value = 0.6;
    const g = ctx.createGain(); g.gain.setValueAtTime(vel * 0.42, t); g.gain.exponentialRampToValueAtTime(0.001, t + (lofi ? 0.16 : 0.2));
    s.connect(bp); bp.connect(g); g.connect(dest); s.start(t, (t * 3.7) % 1.5); s.stop(t + 0.25);
    const o = ctx.createOscillator(); o.frequency.setValueAtTime(220, t); o.frequency.exponentialRampToValueAtTime(160, t + 0.08);
    const og = ctx.createGain(); og.gain.setValueAtTime(vel * 0.25, t); og.gain.exponentialRampToValueAtTime(0.001, t + 0.1); o.connect(og); og.connect(dest); o.start(t); o.stop(t + 0.12);
  }
  function hat(ctx, dest, t, vel = 1, open = false) {
    const s = ctx.createBufferSource(); s.buffer = noise(ctx);
    const hp = ctx.createBiquadFilter(); hp.type = 'highpass'; hp.frequency.value = 7000;
    const g = ctx.createGain(); g.gain.setValueAtTime(vel * 0.14, t); g.gain.exponentialRampToValueAtTime(0.001, t + (open ? 0.22 : 0.045));
    s.connect(hp); hp.connect(g); g.connect(dest); s.start(t, (t * 5.1) % 1.5); s.stop(t + 0.25);
  }
  function brush(ctx, dest, t, vel, len = 0.18) {
    const s = ctx.createBufferSource(); s.buffer = noise(ctx);
    const bp = ctx.createBiquadFilter(); bp.type = 'bandpass'; bp.frequency.value = 4200; bp.Q.value = 0.5;
    const g = ctx.createGain(); g.gain.setValueAtTime(0.001, t); g.gain.linearRampToValueAtTime(vel * 0.1, t + len * 0.4); g.gain.exponentialRampToValueAtTime(0.001, t + len);
    s.connect(bp); bp.connect(g); g.connect(dest); s.start(t, (t * 2.3) % 1.5); s.stop(t + len + 0.02);
  }
  function crackle(ctx, dest, len) {   // vinyl dust under the lo-fi tune
    const n = Math.floor(ctx.sampleRate * len), b = ctx.createBuffer(1, n, ctx.sampleRate), d = b.getChannelData(0);
    let s = 4242; const R = () => (s = (s * 1103515245 + 12345) & 0x7fffffff) / 0x7fffffff;
    for (let i = 0; i < n; i++) { d[i] = (R() - 0.5) * 0.012; if (R() < 0.0006) d[i] += (R() - 0.5) * 0.6; }
    const src = ctx.createBufferSource(); src.buffer = b; const lp = ctx.createBiquadFilter(); lp.type = 'lowpass'; lp.frequency.value = 5000;
    const g = ctx.createGain(); g.gain.value = 0.5; src.connect(lp); lp.connect(g); g.connect(dest); src.start(0);
  }

  /* ---------------- the tunes ---------------- */
  const CH = {   // chord tones (mid register) and the bass root
    Dm7: [[57, 60, 62, 65], 50], G7: [[55, 59, 62, 65], 43], Cmaj7: [[55, 59, 64, 67], 48], Am7: [[55, 57, 60, 64], 45],
    A7: [[55, 57, 61, 64], 45], Fmaj7: [[57, 60, 64, 65], 41], Em7: [[55, 59, 62, 64], 40], F: [[57, 60, 65], 41],
    Bb: [[58, 62, 65], 46], C7: [[55, 58, 60, 64], 48], Dm: [[57, 62, 65], 50], Gm7: [[55, 58, 62, 65], 43],
    Fm6: [[56, 60, 62, 65], 41], C: [[55, 60, 64], 48]
  };
  const n = (s) => s.split(' ').filter(Boolean).map(x => { const [m, d] = x.split(':'); return [+m, +d]; });

  // 1. Fire Escape Beat: lo-fi hip-hop, C major, electric piano over a lazy boom-bap
  const LOFI = {
    name: 'Fire Escape Beat', bpm: 84, beats: 4, style: 'lofi', swing: 0.1,
    A: { ch: 'Dm7 G7 Cmaj7 Am7 Dm7 G7 Cmaj7 A7'.split(' '),
      mel: n('0:1 69:.5 72:.5 74:1 72:1 71:1.5 67:.5 0:2 0:1 64:.5 67:.5 71:1 72:1 69:3 0:1 0:1 69:.5 72:.5 77:1 76:1 74:1.5 71:.5 67:2 72:1 71:.5 67:.5 64:1 67:1 69:3 0:1') },
    B: { ch: 'Fmaj7 Em7 Dm7 Cmaj7 Fmaj7 Em7 Dm7 G7'.split(' '),
      mel: n('77:1 76:.5 74:.5 72:2 76:1 74:.5 72:.5 71:2 74:1 72:.5 69:.5 65:1 69:1 67:4 77:1 79:.5 77:.5 76:1 72:1 74:1 76:.5 74:.5 71:2 69:1 72:1 74:1 77:1 79:3 0:1') },
    form: ['A', 'A', 'B', 'A'], end: 'Cmaj7'
  };
  // 2. Stoop Soul: a soul strut in F, tenor sax over organ, bass and a backbeat
  const SOUL = {
    name: 'Stoop Soul', bpm: 108, beats: 4, style: 'soul', swing: 0,
    A: { ch: 'F Bb F C7 Dm Bb C7 F'.split(' '),
      mel: n('72:1 69:.5 72:.5 74:1 72:1 70:1 74:1 77:2 76:.5 74:.5 72:1 69:1 65:1 67:3 0:1 69:1 72:1 74:1 77:1 74:1.5 72:.5 70:2 72:1 74:.5 76:.5 79:1 76:1 77:3 0:1') },
    B: { ch: 'Bb Bb F F Gm7 C7 F C7'.split(' '),
      mel: n('77:1.5 74:.5 77:1 79:1 81:2 79:1 77:1 76:1 72:1 69:1 72:1 77:2 0:2 74:1 77:1 79:1 82:1 81:1 79:1 76:1 72:1 74:.5 76:.5 77:1 72:1 69:1 72:2 0:2') },
    form: ['A', 'A', 'B', 'A'], end: 'F'
  };
  // 3. Bridge Lights: a late-night ballad in C for piano, upright bass and brushes
  const JAZZ = {
    name: 'Bridge Lights', bpm: 70, beats: 4, style: 'jazz', swing: 0.16,
    A: { ch: 'Cmaj7 A7 Dm7 G7 Em7 A7 Dm7 G7 Cmaj7 C7 Fmaj7 Fm6 Em7 A7 Dm7 G7'.split(' '),
      mel: n('76:1.5 74:.5 72:1 67:1 73:2 69:2 74:1.5 72:.5 69:1 65:1 71:3 67:1 79:1.5 76:.5 74:1 71:1 73:2 76:2 77:1 76:1 74:1 72:1 71:3 0:1 76:1.5 72:.5 76:1 79:1 82:2 79:2 81:1.5 77:.5 72:1 69:1 80:2 77:2 79:1 76:1 72:1 71:1 73:2 69:2 74:1 77:1 76:1 74:1 72:4') },
    form: ['A', 'A'], end: 'Cmaj7'
  };
  const TUNES = [LOFI, SOUL, JAZZ];

  // buildQueue turns a tune into timed note events.
  function buildQueue(tune) {
    const beat = 60 / tune.bpm, ev = [], st = tune.style;
    const sw = x => tune.swing && Math.abs(x % 1 - 0.5) < 1e-6 ? x + tune.swing : x;
    let t = 0.1;
    for (let pass = 0; pass < tune.form.length; pass++) {
      const sec = tune[tune.form[pass]];
      let mt = 0;
      for (const [m, d] of sec.mel) {
        if (m > 0) {
          const a = sw(mt), e = sw(mt + d), dur = Math.max(0.08, (e - a) * beat * 0.92);
          if (st === 'lofi') ev.push({ i: 'rhodes', t: t + a * beat, m, d: dur, v: 0.75 });
          else if (st === 'soul') ev.push({ i: 'sax', t: t + a * beat, m: m - 12, d: dur, v: 0.9 });
          else ev.push({ i: 'piano', t: t + a * beat, m: m + (pass === 1 ? 12 : 0), d: dur, v: 0.8 });
        }
        mt += d;
      }
      sec.ch.forEach((name, k) => {
        const [tones, root] = CH[name], b0 = t + k * tune.beats * beat, bar = k;
        if (st === 'lofi') {
          for (const tn of tones) ev.push({ i: 'rhodes', t: b0 + 0.012 * tones.indexOf(tn), m: tn, d: beat * 3.6, v: 0.4 });
          ev.push({ i: 'bass', t: b0, m: root, d: beat * 1.4, v: 0.95 }); ev.push({ i: 'bass', t: b0 + 2.5 * beat, m: root + (bar % 2 ? 7 : 0), d: beat * 1.2, v: 0.8 });
          for (const b of [0, 1.75, 2.5]) ev.push({ i: 'kick', t: b0 + b * beat, v: b ? 0.75 : 0.9 });
          for (const b of [1, 3]) ev.push({ i: 'snare', t: b0 + b * beat, v: 0.75, lofi: true });
          for (let h = 0; h < 8; h++) ev.push({ i: 'hat', t: b0 + sw(h / 2) * beat, v: h % 2 ? 0.45 : 0.7 });
        } else if (st === 'soul') {
          for (const b of [1, 3]) for (const tn of tones) ev.push({ i: 'organ', t: b0 + b * beat, m: tn, d: beat * 0.55, v: 0.8 });
          for (const tn of tones) ev.push({ i: 'organ', t: b0, m: tn - 12, d: beat * 3.8, v: 0.25 });
          for (const [b, m] of [[0, root], [1.5, root + 12], [2, root + 7], [3.5, root + (bar % 2 ? 5 : 9)]]) ev.push({ i: 'bass', t: b0 + b * beat, m, d: beat * 0.45, v: 0.95 });
          for (const b of [0, 2, 2.5]) ev.push({ i: 'kick', t: b0 + b * beat, v: 0.8 });
          for (const b of [1, 3]) ev.push({ i: 'snare', t: b0 + b * beat, v: 0.85 });
          for (let h = 0; h < 8; h++) ev.push({ i: 'hat', t: b0 + h * beat / 2, v: h % 2 ? 0.5 : 0.75, open: h === 7 });
        } else {
          for (const b of [0, 1.5 + tune.swing]) for (const tn of tones) ev.push({ i: 'piano', t: b0 + b * beat, m: tn, d: beat * (b ? 2 : 1.2), v: 0.32 });
          const walk = [root, root + 4, root + 7, root + (k % 2 ? 6 : 10)];
          walk.forEach((m, b) => ev.push({ i: 'upright', t: b0 + b * beat, m: m - 12 + 12, d: beat * 0.9, v: 0.9 }));
          for (let b = 0; b < 4; b++) { ev.push({ i: 'brush', t: b0 + b * beat, v: 0.9, len: beat * 0.6 }); if (b % 2) ev.push({ i: 'brush', t: b0 + (b + 0.5 + tune.swing) * beat, v: 0.5, len: 0.08 }); }
        }
      });
      t += sec.ch.length * tune.beats * beat;
    }
    const [tones, root] = CH[tune.end];
    ev.push({ i: st === 'jazz' ? 'upright' : 'bass', t, m: root, d: beat * 3, v: 1 });
    const fin = st === 'lofi' ? 'rhodes' : st === 'soul' ? 'organ' : 'piano';
    for (const tn of tones) ev.push({ i: fin, t, m: tn, d: beat * 3, v: 0.5 });
    ev.push({ i: st === 'soul' ? 'sax' : fin, t, m: tones[tones.length - 1] + (st === 'soul' ? 0 : 12), d: beat * 3, v: 0.7 });
    if (st !== 'jazz') ev.push({ i: 'kick', t, v: 0.9 });
    return { ev, len: t + beat * 3 + 1.4 };
  }

  function renderTune(ctx, dest, tune) {
    const jazz = tune.style === 'jazz';
    const rv = reverb(ctx, jazz ? 2.4 : 1.4, jazz ? 0.28 : 0.16); rv.out.connect(dest);
    const bus = ctx.createGain(); bus.gain.value = 0.85; bus.connect(rv.inp);
    const lo = ctx.createBiquadFilter(); lo.type = 'lowpass'; lo.frequency.value = tune.style === 'lofi' ? 5200 : 16000; lo.connect(bus);
    const q = buildQueue(tune);
    if (tune.style === 'lofi') crackle(ctx, dest, q.len);
    for (const e of q.ev) {
      if (e.i === 'rhodes') rhodes(ctx, lo, e.t, e.m, e.d, e.v);
      else if (e.i === 'organ') organ(ctx, lo, e.t, e.m, e.d, e.v);
      else if (e.i === 'sax') sax(ctx, lo, e.t, e.m, e.d, e.v);
      else if (e.i === 'bass') bass(ctx, lo, e.t, e.m, e.d, e.v);
      else if (e.i === 'upright') upright(ctx, lo, e.t, e.m, e.d, e.v);
      else if (e.i === 'piano') piano(ctx, lo, e.t, e.m, e.d, e.v);
      else if (e.i === 'kick') kick(ctx, lo, e.t, e.v);
      else if (e.i === 'snare') snare(ctx, lo, e.t, e.v, e.lofi);
      else if (e.i === 'hat') hat(ctx, lo, e.t, e.v, e.open);
      else if (e.i === 'brush') brush(ctx, lo, e.t, e.v, e.len);
    }
    return q.len;
  }

  /* ---------------- sound effects ---------------- */
  function flick(ctx, dest, t, vel = 1, f = 2600, len = 0.06) {   // one card snapping through the air
    const s = ctx.createBufferSource(); s.buffer = noise(ctx);
    const bp = ctx.createBiquadFilter(); bp.type = 'bandpass'; bp.frequency.setValueAtTime(f, t); bp.frequency.exponentialRampToValueAtTime(f * 0.5, t + len); bp.Q.value = 1.2;
    const g = ctx.createGain(); g.gain.setValueAtTime(0, t); g.gain.linearRampToValueAtTime(vel * 0.6, t + 0.004); g.gain.exponentialRampToValueAtTime(0.001, t + len);
    s.connect(bp); bp.connect(g); g.connect(dest); s.start(t, (t * 7.3) % 1.5); s.stop(t + len + 0.02);
  }
  function thump(ctx, dest, t, vel = 1) {   // a card landing on marble
    flick(ctx, dest, t, vel * 0.8, 1800, 0.05);
    const o = ctx.createOscillator(); o.frequency.setValueAtTime(180, t); o.frequency.exponentialRampToValueAtTime(70, t + 0.06);
    const g = ctx.createGain(); g.gain.setValueAtTime(vel * 0.25, t); g.gain.exponentialRampToValueAtTime(0.001, t + 0.08);
    o.connect(g); g.connect(dest); o.start(t); o.stop(t + 0.1);
  }
  function bell(ctx, dest, t, m, vel = 1, len = 1.2) {   // a little chime
    for (const [k, a] of [[1, 1], [2.76, 0.35], [5.4, 0.15]]) {
      const o = ctx.createOscillator(); o.type = 'sine'; o.frequency.value = mtof(m) * k;
      const g = ctx.createGain(); g.gain.setValueAtTime(0, t); g.gain.linearRampToValueAtTime(vel * 0.16 * a, t + 0.004); g.gain.exponentialRampToValueAtTime(0.0005, t + len / k);
      o.connect(g); g.connect(dest); o.start(t); o.stop(t + len);
    }
  }
  function snap(ctx, dest, t, vel = 1) {   // a finger snap
    const s = ctx.createBufferSource(); s.buffer = noise(ctx);
    const bp = ctx.createBiquadFilter(); bp.type = 'bandpass'; bp.frequency.value = 2600; bp.Q.value = 2;
    const g = ctx.createGain(); g.gain.setValueAtTime(vel * 0.9, t); g.gain.exponentialRampToValueAtTime(0.001, t + 0.05);
    s.connect(bp); bp.connect(g); g.connect(dest); s.start(t, 0.3); s.stop(t + 0.06);
  }
  const fx = {
    shuffle(ctx, d) { let t = 0.02; for (let i = 0; i < 26; i++) { flick(ctx, d, t, 0.45 + 0.2 * Math.sin(i), 3200 + (i % 5) * 200, 0.035); t += 0.018 + (i % 3) * 0.004; } thump(ctx, d, t + 0.05, 0.8); return t + 0.3; },
    deal(ctx, d) { flick(ctx, d, 0.01, 0.8, 3000, 0.07); return 0.15; },
    play(ctx, d) { flick(ctx, d, 0.01, 0.5, 2400, 0.05); thump(ctx, d, 0.05, 0.9); return 0.2; },
    hover(ctx, d) { const o = ctx.createOscillator(); o.frequency.value = 2200; const g = ctx.createGain(); g.gain.setValueAtTime(0.03, 0.005); g.gain.exponentialRampToValueAtTime(0.0005, 0.04); o.connect(g); g.connect(d); o.start(0.005); o.stop(0.05); return 0.06; },
    tick(ctx, d) { flick(ctx, d, 0.005, 0.5, 4200, 0.025); const o = ctx.createOscillator(); o.frequency.value = 1500; const g = ctx.createGain(); g.gain.setValueAtTime(0.08, 0.005); g.gain.exponentialRampToValueAtTime(0.0005, 0.05); o.connect(g); g.connect(d); o.start(0.005); o.stop(0.06); return 0.08; },
    slide(ctx, d) { const s = ctx.createBufferSource(); s.buffer = noise(ctx); const bp = ctx.createBiquadFilter(); bp.type = 'bandpass'; bp.frequency.setValueAtTime(1200, 0.01); bp.frequency.linearRampToValueAtTime(2600, 0.2); bp.Q.value = 0.8; const g = ctx.createGain(); g.gain.setValueAtTime(0, 0.01); g.gain.linearRampToValueAtTime(0.25, 0.06); g.gain.exponentialRampToValueAtTime(0.001, 0.24); s.connect(bp); bp.connect(g); g.connect(d); s.start(0.01); s.stop(0.26); return 0.3; },
    sweep(ctx, d) { const s = ctx.createBufferSource(); s.buffer = noise(ctx); const bp = ctx.createBiquadFilter(); bp.type = 'bandpass'; bp.frequency.setValueAtTime(800, 0.01); bp.frequency.exponentialRampToValueAtTime(3000, 0.28); bp.Q.value = 0.7; const g = ctx.createGain(); g.gain.setValueAtTime(0, 0.01); g.gain.linearRampToValueAtTime(0.3, 0.1); g.gain.exponentialRampToValueAtTime(0.001, 0.34); s.connect(bp); bp.connect(g); g.connect(d); s.start(0.01); s.stop(0.36); return 0.4; },
    trick(ctx, d) { snap(ctx, d, 0.01, 0.8); rhodes(ctx, d, 0.03, 79, 0.25, 0.7); rhodes(ctx, d, 0.12, 84, 0.4, 0.6); return 1.2; },
    bid(ctx, d) {   // a wood-block knock on the table
      for (const [t, f] of [[0.01, 880], [0.1, 1180]]) { const o = ctx.createOscillator(); o.frequency.value = f; const g = ctx.createGain(); g.gain.setValueAtTime(0.35, t); g.gain.exponentialRampToValueAtTime(0.001, t + 0.07); o.connect(g); g.connect(d); o.start(t); o.stop(t + 0.08); flick(ctx, d, t, 0.25, 3600, 0.02); }
      return 0.3;
    },
    nil(ctx, d) {   // a slide whistle up: going for nothing
      const o = ctx.createOscillator(); o.type = 'sine'; o.frequency.setValueAtTime(520, 0.02); o.frequency.exponentialRampToValueAtTime(1500, 0.5);
      const vib = ctx.createOscillator(); vib.frequency.value = 7; const vg = ctx.createGain(); vg.gain.value = 25; vib.connect(vg); vg.connect(o.detune);
      const g = ctx.createGain(); g.gain.setValueAtTime(0, 0.02); g.gain.linearRampToValueAtTime(0.25, 0.08); g.gain.setValueAtTime(0.25, 0.45); g.gain.exponentialRampToValueAtTime(0.001, 0.62);
      o.connect(g); g.connect(d); o.start(0.02); o.stop(0.65); vib.start(0.02); vib.stop(0.65); return 0.7;
    },
    nilmade(ctx, d) { [72, 76, 79, 84, 88].forEach((m, i) => rhodes(ctx, d, 0.02 + i * 0.08, m, 0.6, 0.75)); bell(ctx, d, 0.42, 96, 0.6, 1.4); return 1.8; },
    bust(ctx, d) {  // a sad trombone
      [[62, 0.28, 0], [61, 0.28, 0], [60, 0.28, 0], [59, 0.9, -1]].reduce((t, [m, dur, b]) => { sax(ctx, d, t, m - 12, dur * 0.92, 0.95, b); return t + dur; }, 0.02);
      return 2;
    },
    made(ctx, d) { rhodes(ctx, d, 0.02, 72, 0.3, 0.8); rhodes(ctx, d, 0.16, 79, 0.6, 0.8); rhodes(ctx, d, 0.16, 76, 0.6, 0.5); return 1.2; },
    set(ctx, d) {   // a deflating organ "womp womp"
      [[67, 0.3, -1], [60, 0.7, -2]].reduce((t, [m, dur, b]) => { for (const k of [0, 3, 7]) organ(ctx, d, t, m + k - 12, dur, 0.9); sax(ctx, d, t, m - 12, dur, 0.6, b); return t + dur + 0.05; }, 0.02);
      return 1.4;
    },
    bags(ctx, d) {  // a thud, and a sack of loose change
      kick(ctx, d, 0.02, 0.9);
      for (let i = 0; i < 14; i++) bell(ctx, d, 0.1 + i * 0.025 + (i % 3) * 0.01, 96 + (i * 5) % 9, 0.25, 0.25);
      return 0.9;
    },
    trump(ctx, d) { // a horn stab: the spade comes down
      for (const m of [55, 60, 64, 70]) sax(ctx, d, 0.02, m - 12, 0.22, 0.8);
      kick(ctx, d, 0.02, 0.8); snare(ctx, d, 0.02, 0.5);
      return 0.6;
    },
    broken(ctx, d) { // a record scratch and a bass drop: spades are out
      const s = ctx.createBufferSource(); s.buffer = noise(ctx); const bp = ctx.createBiquadFilter(); bp.type = 'bandpass'; bp.Q.value = 3;
      bp.frequency.setValueAtTime(600, 0.02); bp.frequency.exponentialRampToValueAtTime(2600, 0.12); bp.frequency.exponentialRampToValueAtTime(500, 0.24); bp.frequency.exponentialRampToValueAtTime(2200, 0.32);
      const g = ctx.createGain(); g.gain.setValueAtTime(0, 0.02); g.gain.linearRampToValueAtTime(0.6, 0.05); g.gain.setValueAtTime(0.6, 0.3); g.gain.exponentialRampToValueAtTime(0.001, 0.36);
      s.connect(bp); bp.connect(g); g.connect(d); s.start(0.02); s.stop(0.38);
      const o = ctx.createOscillator(); o.frequency.setValueAtTime(110, 0.38); o.frequency.exponentialRampToValueAtTime(38, 1.1);
      const og = ctx.createGain(); og.gain.setValueAtTime(0.6, 0.38); og.gain.exponentialRampToValueAtTime(0.001, 1.2); o.connect(og); og.connect(d); o.start(0.38); o.stop(1.25);
      return 1.3;
    },
    win(ctx, d) {   // a little horn fanfare
      const seq = [[67, .14], [72, .14], [76, .14], [79, .45], [76, .14], [79, .7]];
      let t = 0.02; for (const [m, dur] of seq) { sax(ctx, d, t, m - 12, dur * 0.95, 0.9); rhodes(ctx, d, t, m + 12, dur, 0.4); t += dur; }
      for (const m of [48, 55, 64, 67]) organ(ctx, d, t - 0.7, m, 0.8, 0.8);
      kick(ctx, d, t - 0.7, 0.8); bell(ctx, d, t - 0.7, 96, 0.6, 1.6);
      return t + 1.2;
    },
    lose(ctx, d) {
      const seq = [[67, .3], [65, .3], [64, .3], [60, .9]];
      let t = 0.02; for (const [m, dur] of seq) { sax(ctx, d, t, m - 12, dur * 0.95, 0.7); t += dur; }
      for (const m of [48, 55, 60]) organ(ctx, d, t - 0.9, m, 0.9, 0.6);
      return t + 0.8;
    }
  };

  return { fx, TUNES, renderTune, buildQueue, reverb };
})();
