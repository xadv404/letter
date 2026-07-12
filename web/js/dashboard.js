(function () {
  var logBox = document.getElementById("log-box");
  var dorkPreview = document.getElementById("dork-preview");
  var domainsBody = document.getElementById("domains-body");
  var decisionsBox = document.getElementById("decisions-box");
  var chartCanvas = document.getElementById("discovery-chart");
  var goroutinesHint = document.getElementById("goroutines-hint");

  var btnStart = document.getElementById("btn-start");
  var btnPause = document.getElementById("btn-pause");
  var btnResume = document.getElementById("btn-resume");
  var btnStop = document.getElementById("btn-stop");
  var fileInput = document.getElementById("domains-file");
  var domainLabel = document.getElementById("domain-file-label");

  function setRunning(on) {
    btnStart.disabled = on;
    btnPause.disabled = !on;
    btnResume.disabled = !on;
    btnStop.disabled = !on;
  }

  function appendLog(line) {
    logBox.textContent = (logBox.textContent ? logBox.textContent + "\n" : "") + line;
    logBox.scrollTop = logBox.scrollHeight;
  }

  function drawChart(series) {
    if (!chartCanvas || !series || !series.length) return;
    var ctx = chartCanvas.getContext("2d");
    var w = chartCanvas.width;
    var h = chartCanvas.height;
    var pad = { l: 36, r: 12, t: 12, b: 22 };
    ctx.clearRect(0, 0, w, h);

    var maxVal = 1;
    series.forEach(function (p) {
      maxVal = Math.max(maxVal, p.keywords || 0, p.params || 0);
    });

    function xAt(i) {
      return pad.l + (i / Math.max(1, series.length - 1)) * (w - pad.l - pad.r);
    }
    function yAt(v) {
      return h - pad.b - (v / maxVal) * (h - pad.t - pad.b);
    }

    function fillArea(key, color) {
      ctx.beginPath();
      series.forEach(function (p, i) {
        var x = xAt(i);
        var y = yAt(p[key] || 0);
        if (i === 0) ctx.moveTo(x, y);
        else ctx.lineTo(x, y);
      });
      ctx.lineTo(xAt(series.length - 1), h - pad.b);
      ctx.lineTo(xAt(0), h - pad.b);
      ctx.closePath();
      ctx.fillStyle = color;
      ctx.fill();
    }

    fillArea("keywords", "rgba(103,232,249,0.25)");
    fillArea("params", "rgba(252,211,77,0.2)");

    function strokeLine(key, color) {
      ctx.beginPath();
      series.forEach(function (p, i) {
        var x = xAt(i);
        var y = yAt(p[key] || 0);
        if (i === 0) ctx.moveTo(x, y);
        else ctx.lineTo(x, y);
      });
      ctx.strokeStyle = color;
      ctx.lineWidth = 2;
      ctx.stroke();
    }

    strokeLine("keywords", "#67e8f9");
    strokeLine("params", "#fcd34d");

    ctx.fillStyle = "rgba(148,163,184,0.7)";
    ctx.font = "10px JetBrains Mono, monospace";
    ctx.fillText("kw", pad.l, pad.t + 8);
    ctx.fillText("params", pad.l + 28, pad.t + 8);
    ctx.fillText(String(maxVal), 4, pad.t + 10);
  }

  function renderDecisions(decisions) {
    if (!decisions || !decisions.length) {
      decisionsBox.textContent = "Aucune décision encore…";
      return;
    }
    var lines = decisions.slice(-100).reverse().map(function (d) {
      var flag = d.accepted ? "ACCEPT" : "REJECT";
      return "[" + flag + "] " + d.param + " score=" + d.score + " " + d.tier + " — " + d.reason;
    });
    decisionsBox.textContent = lines.join("\n");
    decisionsBox.scrollTop = 0;
  }

  function applySnapshot(s) {
    if (!s) return;
    setRunning(!!s.running);
    var statusEl = document.getElementById("status-detail");
    if (s.running) {
      statusEl.textContent = (s.phase_label || "En cours") + " · " + (s.throttle || "NORMAL");
      statusEl.className = "page-subtitle status-running";
    } else if (s.elapsed && s.elapsed !== "0s") {
      statusEl.textContent = "Terminé";
      statusEl.className = "page-subtitle status-done";
    } else {
      statusEl.textContent = "En attente";
      statusEl.className = "page-subtitle";
    }
    document.getElementById("stat-elapsed").textContent = s.elapsed || "0s";
    document.getElementById("stat-kw").textContent = s.keywords || 0;
    document.getElementById("stat-params").textContent = s.params || 0;
    document.getElementById("stat-filter").textContent =
      (s.accepted || 0) + " acceptés / " + (s.rejected || 0) + " rejetés";
    document.getElementById("stat-cpu").textContent = s.cpu != null ? s.cpu.toFixed(1) + "%" : "—";
    document.getElementById("stat-ram").textContent =
      (s.ram != null ? s.ram.toFixed(1) + "% RAM" : "") +
      " · " + (s.throttle || "NORMAL") +
      " · " + (s.workers || 0) + " workers";

    if (goroutinesHint) {
      goroutinesHint.textContent = s.goroutines ? "(goroutines: " + s.goroutines + ")" : "";
    }

    if (s.phase >= 4 && s.dork_preview) {
      dorkPreview.textContent = s.dork_preview;
    } else if (s.running) {
      dorkPreview.textContent = "Phase " + (s.phase || 1) + "/4 — dorks en phase 4 uniquement…";
    } else if (!s.dork_preview) {
      dorkPreview.textContent = "En attente…";
    }
    if (s.domain_file) domainLabel.textContent = "Fichier : " + s.domain_file;

    drawChart(s.timeseries || []);
    renderDecisions(s.decisions || []);

    domainsBody.innerHTML = "";
    (s.domains || []).forEach(function (d) {
      var tr = document.createElement("tr");
      tr.innerHTML =
        "<td>" + d.Domain + "</td><td>" + d.Pages + "</td><td>" + d.Errors + "</td><td>" +
        (d.Finished ? "done" : "running") + "</td>";
      domainsBody.appendChild(tr);
    });
  }

  function onEvent(ev) {
    if (ev.type === "init" && ev.data) {
      if (ev.data.logs) logBox.textContent = ev.data.logs.join("\n");
      applySnapshot(ev.data.snapshot);
      if (ev.data.config && ev.data.config.domain_file) {
        domainLabel.textContent = "Fichier : " + ev.data.config.domain_file;
      }
    }
    if (ev.type === "snapshot") applySnapshot(ev.data);
    if (ev.type === "log") appendLog(ev.line);
    if (ev.type === "dorks_ready") {
      if (ev.saved) appendLog("Fichier enregistré via l'explorateur Windows");
      else if (ev.cancelled) appendLog("Enregistrement annulé");
    }
  }

  LetterAPI.connectEvents(onEvent);

  fileInput.addEventListener("change", function () {
    var f = fileInput.files[0];
    if (!f) return;
    LetterAPI.uploadDomains(f)
      .then(function (r) {
        domainLabel.textContent = "Fichier : " + r.domain_file;
        appendLog("Import : " + f.name);
      })
      .catch(function (e) { alert(e.message); });
  });

  btnStart.addEventListener("click", function () {
    LetterAPI.start().catch(function (e) { alert(e.message); });
  });
  btnPause.addEventListener("click", function () { LetterAPI.pause().catch(function (e) { alert(e.message); }); });
  btnResume.addEventListener("click", function () { LetterAPI.resume().catch(function (e) { alert(e.message); }); });
  btnStop.addEventListener("click", function () { LetterAPI.stop().catch(function (e) { alert(e.message); }); });
})();
