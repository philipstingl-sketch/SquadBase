'use strict';
// SQUAD BASE 4.7.0 comfort preferences: all app settings are stored locally.
(() => {
  const $ = (id) => document.getElementById(id);
  const controls = {
    startup: $('comfort-startup'),
    tray: $('comfort-tray'),
    theme: $('comfort-theme-mode'),
    save: $('comfort-save'),
    hide: $('comfort-hide-now'),
    status: $('comfort-status'),
  };
  let settings = { startWithWindows: false, minimizeToTray: false, themeMode: 'dark' };
  let windowsAvailable = false;

  async function api(path, payload) {
    const options = payload === undefined ? {} : {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Squad-Base': '1' },
      body: JSON.stringify(payload),
    };
    const response = await fetch('api/' + path, options);
    let data = {};
    try { data = await response.json(); } catch { /* server may close the window */ }
    if (!response.ok) throw new Error(data.error || ('HTTP ' + response.status));
    return data;
  }

  function status(message, error = false) {
    if (!controls.status) return;
    controls.status.textContent = message;
    controls.status.classList.toggle('comfort-error', error);
  }

  function applyTheme(mode) {
    settings.themeMode = ['dark', 'light', 'system'].includes(mode) ? mode : 'dark';
    const prefersLight = window.matchMedia?.('(prefers-color-scheme: light)').matches === true;
    const applied = settings.themeMode === 'system' ? (prefersLight ? 'light' : 'dark') : settings.themeMode;
    document.documentElement.dataset.themeMode = applied;
    document.documentElement.dataset.themePreference = settings.themeMode;
  }

  function updateControls() {
    if (controls.startup) controls.startup.checked = !!settings.startWithWindows;
    if (controls.tray) controls.tray.checked = !!settings.minimizeToTray;
    if (controls.theme) controls.theme.value = settings.themeMode;
    if (controls.startup) controls.startup.disabled = !windowsAvailable;
    if (controls.tray) controls.tray.disabled = !windowsAvailable;
    if (controls.hide) controls.hide.disabled = !windowsAvailable;
    applyTheme(settings.themeMode);
  }

  async function load() {
    try {
      const data = await api('comfort');
      windowsAvailable = !!data.windows;
      settings = { ...settings, ...(data.settings || {}) };
      updateControls();
      status(windowsAvailable
        ? 'Lokale Komfort-Einstellungen geladen. Änderungen gelten nach dem Speichern.'
        : 'Windows-Autostart und Infobereich sind nur in der Windows-App verfügbar.');
    } catch (error) {
      status('Einstellungen konnten nicht geladen werden: ' + error.message, true);
    }
  }

  controls.theme?.addEventListener('change', () => applyTheme(controls.theme.value));
  window.matchMedia?.('(prefers-color-scheme: light)').addEventListener?.('change', () => {
    if (settings.themeMode === 'system') applyTheme('system');
  });

  controls.save?.addEventListener('click', async () => {
    if (controls.save.disabled) return;
    controls.save.disabled = true;
    const next = {
      startWithWindows: !!controls.startup?.checked,
      minimizeToTray: !!controls.tray?.checked,
      themeMode: controls.theme?.value || 'dark',
    };
    try {
      const data = await api('comfort', { action: 'configure', settings: next });
      settings = { ...next, ...(data.settings || {}) };
      updateControls();
      status(next.minimizeToTray
        ? 'Gespeichert. Das Tray-Symbol ist aktiv; zum Beenden das Tray-Menü öffnen.'
        : 'Komfort-Einstellungen gespeichert.');
      window.notify?.('Komfort-Einstellungen gespeichert.');
    } catch (error) {
      status('Speichern fehlgeschlagen: ' + error.message, true);
      window.notify?.(error.message, true);
      await load();
    } finally {
      controls.save.disabled = false;
    }
  });

  controls.hide?.addEventListener('click', async () => {
    controls.hide.disabled = true;
    try {
      if (!settings.minimizeToTray) {
        throw new Error('Aktiviere zuerst „Im Infobereich behalten“ und speichere die Einstellung.');
      }
      await api('comfort', { action: 'hide' });
    } catch (error) {
      status(error.message, true);
      window.notify?.(error.message, true);
    } finally {
      controls.hide.disabled = !windowsAvailable;
    }
  });

  // Full backup export. The server only returns the validated settings schema,
  // never tokens, OAuth credentials, friend chat history, or diagnostics.
  $('v4-config-export')?.addEventListener('click', async (event) => {
    const button = event.currentTarget;
    if (button.disabled) return;
    button.disabled = true;
    try {
      const backup = await api('diagnostics', { action: 'export' });
      const blob = new Blob([JSON.stringify(backup, null, 2) + '\n'], { type: 'application/json' });
      const href = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = href;
      const stamp = new Date().toISOString().replace(/[:.]/g, '-');
      link.download = 'SQUAD_BASE_Einstellungen_' + stamp + '.json';
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.setTimeout(() => URL.revokeObjectURL(href), 1000);
      window.notify?.('Backup als JSON exportiert. Bewahre die Datei privat auf.');
    } catch (error) {
      window.notify?.('Backup-Export fehlgeschlagen: ' + error.message, true);
    } finally {
      button.disabled = false;
    }
  });

  const importTrigger = $('v4-config-import-trigger');
  const importFile = $('v4-config-import-file');
  importTrigger?.addEventListener('click', () => importFile?.click());
  importFile?.addEventListener('change', async () => {
    const file = importFile.files?.[0];
    if (!file) return;
    if (file.size > 2 * 1024 * 1024) {
      window.notify?.('Die Sicherung darf höchstens 2 MB groß sein.', true);
      importFile.value = '';
      return;
    }
    if (!window.confirm('Diese Sicherung importieren? Die aktuellen Einstellungen werden vorher als Rückfallsicherung gespeichert. Nach dem Import SQUAD BASE schließen und erneut öffnen.')) {
      importFile.value = '';
      return;
    }
    importTrigger.disabled = true;
    try {
      const backup = JSON.parse(await file.text());
      await api('diagnostics', { action: 'restore-upload', backup });
      window.notify?.('Sicherung importiert. Bitte SQUAD BASE vollständig schließen und neu starten.');
      status('Sicherung importiert. Änderungen werden nach einem Neustart vollständig wirksam.');
      await load();
    } catch (error) {
      window.notify?.('Import fehlgeschlagen: ' + error.message, true);
      status('Import fehlgeschlagen: ' + error.message, true);
    } finally {
      importTrigger.disabled = false;
      importFile.value = '';
    }
  });

  load();
})();
