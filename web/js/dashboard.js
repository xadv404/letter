(function () {
  var logBox = document.getElementById("log-box");
  var dorkPreview = document.getElementById("dork-preview");
  var domainsBody = document.getElementById("domains-body");
  var running = false;

  var btnStart = document.getElementById("btn-start");
  var btnPause = document.getElementById("btn-pause");
  var btnResume = document.getElementById("btn-resume");
  var btnStop = document.getElementById("btn-stop");
  var fileInput = document.getElementById("domains-file");
  var domainLabel = document.getElementById("domain-file-label");

  function setRunning(on) {
    running = on;
    btnStart.disabled = on;
    btnPause.disabled = !on;
    btnResume.disabled = !on;
    btnStop.disabled = !on;
    document.getElementById("status-text").textContent = on ? "Running" : "Idle";
  }

  function appendLog(line) {
    logBox.textContent = (logBox.textContent ? logBox.textContent + "\n" : "") + line;
    logBox.scrollTop = logBox.scrollHeight;
  }

  function applySnapshot(s) {
    if (!s) return;
    setRunning(!!s.running);
    document.getElementById("phase-label").textContent = s.phase_label || "En attente";
    document.getElementById("stat-phase").textContent = s.phase ? s.phase + "/4" : "—";
    document.getElementById("stat-elapsed").textContent = s.elapsed || "0s";
    document.getElementById("stat-kw").textContent = s.keywords || 0;
    document.getElementById("stat-params").textContent = s.params || 0;
    document.getElementById("stat-filter").textContent = (s.accepted || 0) + "/" + (s.rejected || 0) + " filtrés";
    document.getElementById("stat-cpu").textContent = s.cpu != null ? s.cpu.toFixed(1) + "%" : "—";
    document.getElementById("stat-ram").textContent = (s.ram != null ? s.ram.toFixed(1) + "% RAM" : "") + " · " + (s.throttle || "NORMAL");

    document.querySelectorAll(".pipeline-step").forEach(function (el) {
      var p = parseInt(el.getAttribute("data-phase"), 10);
      el.classList.remove("done", "active");
      if (s.phase > p) el.classList.add("done");
      else if (s.phase === p && s.running) el.classList.add("active");
      else if (s.phase === p && !s.running && p === 4) el.classList.add("done");
    });

    if (s.dork_preview) dorkPreview.textContent = s.dork_preview;
    if (s.domain_file) domainLabel.textContent = "Fichier : " + s.domain_file;

    domainsBody.innerHTML = "";
    (s.domains || []).forEach(function (d) {
      var tr = document.createElement("tr");
      tr.innerHTML = "<td>" + d.Domain + "</td><td>" + d.Pages + "</td><td>" + d.Errors + "</td><td>" + (d.Finished ? "done" : "running") + "</td>";
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
      .then(function (r) { domainLabel.textContent = "Fichier : " + r.domain_file; appendLog("Import : " + f.name); })
      .catch(function (e) { alert(e.message); });
  });

  btnStart.addEventListener("click", function () {
    LetterAPI.start().catch(function (e) { alert(e.message); });
  });
  btnPause.addEventListener("click", function () { LetterAPI.pause().catch(function (e) { alert(e.message); }); });
  btnResume.addEventListener("click", function () { LetterAPI.resume().catch(function (e) { alert(e.message); }); });
  btnStop.addEventListener("click", function () { LetterAPI.stop().catch(function (e) { alert(e.message); }); });
})();
