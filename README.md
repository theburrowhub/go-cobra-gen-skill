# go-cobra-gen-skill

Añade un comando `gen-skill` a cualquier CLI basada en [Cobra](https://github.com/spf13/cobra) que genera un **Agent Skill** compatible con [agentskills.io](https://agentskills.io) para Claude Code, Cursor, OpenAI Codex y otros agentes.

## Instalación

```sh
go get github.com/theburrowhub/go-cobra-gen-skill
```

## Uso

```go
import cobragenskill "github.com/theburrowhub/go-cobra-gen-skill"

func main() {
    root := &cobra.Command{Use: "mytool", Short: "Mi CLI"}
    // ... tus comandos ...

    cobragenskill.RegisterCommand(root,
        cobragenskill.WithVersion("1.0.0"),
        cobragenskill.WithLicense("MIT"),
    )

    root.Execute()
}
```

Esto añade el subcomando `gen-skill` a tu CLI:

```sh
mytool gen-skill                    # genera con Claude, instala en el proyecto
mytool gen-skill --agent codex      # usa Codex en lugar de Claude
mytool gen-skill --scope global     # instala globalmente (~/.claude/skills/ etc.)
mytool gen-skill --no-ai            # sin IA, genera desde el texto de ayuda
mytool gen-skill --dry-run          # imprime el SKILL.md sin escribir nada
```

## Cómo funciona

1. **Recopila** el texto de ayuda de todo el árbol de comandos Cobra.
2. **Genera** el `SKILL.md` invocando al agente en modo headless. Si el binario no está disponible o falla, usa el texto de ayuda como fallback.
3. **Instala** el skill en el directorio nativo del agente elegido y siempre en `.agents/skills/` (convención cross-client).

### Directorios de instalación

| `--agent` | `--scope project` | `--scope global` |
|---|---|---|
| `claude` | `.claude/skills/<name>/` | `~/.claude/skills/<name>/` |
| `codex` | `.codex/skills/<name>/` | `~/.codex/skills/<name>/` |
| `gemini` | `.gemini/skills/<name>/` | `~/.gemini/skills/<name>/` |
| todos | `.agents/skills/<name>/` | `~/.agents/skills/<name>/` |

`.agents/skills/` se incluye siempre independientemente del agente seleccionado.

## Flags

| Flag | Valores | Default |
|---|---|---|
| `--agent` | `claude` \| `codex` \| `gemini` | `claude` |
| `--scope` | `project` \| `global` | `project` |
| `--name` | string | nombre del binario |
| `--no-ai` | — | `false` |
| `--dry-run` | — | `false` |

## Opciones de registro

```go
cobragenskill.RegisterCommand(root,
    cobragenskill.WithSkillName("mi-tool"),       // sobreescribe el nombre del skill
    cobragenskill.WithDescription("..."),          // descripción personalizada
    cobragenskill.WithAgent(cobragenskill.AgentCodex), // agente por defecto
    cobragenskill.WithAgentBin("/usr/local/bin/claude"), // ruta custom al binario
    cobragenskill.WithVersion("1.0.0"),
    cobragenskill.WithLicense("MIT"),
    cobragenskill.WithMetadata("author", "mi-org"),
)
```

## Ejemplo

Ver [`example/main.go`](example/main.go) para una CLI de ejemplo completa.

```sh
go run ./example gen-skill --dry-run --no-ai
```
