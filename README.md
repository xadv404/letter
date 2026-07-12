# Letter Recon

Outil de recon SQLi : crawl multi-domaines → keywords → scoring params → dorks notés.

## Lancer (Windows)

1. Télécharge `letter.exe` depuis [Releases](https://github.com/xadv404/letter/releases)
2. Double-clique `letter.exe`
3. **Prérequis** : [Microsoft Edge WebView2 Runtime](https://go.microsoft.com/fwlink/p/?LinkId=2124703)  
   Sans WebView2, l'app s'ouvre dans ton navigateur par défaut avec un message explicatif.

## Lancer (dev / Linux / macOS)

```bash
chmod +x scripts/dev.sh
./scripts/dev.sh
```

Ou manuellement :

```bash
go generate ./internal/dashboard/...
go run ./cmd/letter
```

Le dashboard s'ouvre dans le navigateur. `Ctrl+C` pour quitter.

## Build Windows (.exe)

```bash
./scripts/build.sh
# → dist/letter.exe
```

## Utilisation

1. **Upload Domains** — fichier `.txt` (un domaine par ligne)
2. **Start** — crawl → export dans `./output/`
3. **Pause / Resume** — sans perte de données
4. Fichiers générés : `dorks.txt` (notés), `keywords.txt`, `params.txt`, `dorktypes.txt`

## Dépannage

| Problème | Solution |
|----------|----------|
| Double-clic, rien ne se passe | Installe WebView2 (lien ci-dessus) ou lance depuis `cmd` pour voir l'erreur |
| `index.html manquant` | `go generate ./internal/dashboard/...` |
| Port déjà utilisé | Ferme les autres instances de Letter Recon |
