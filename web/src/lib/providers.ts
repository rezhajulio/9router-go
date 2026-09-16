// Auto-generated provider catalog matching upstream 9router
export interface ProviderCatalogItem {
  id: string
  name: string
  category: 'oauth' | 'free' | 'freeTier' | 'apikey' | 'webCookie' | 'custom'
  alias: string
  color: string
  icon: string
  noAuth?: boolean
}

export const PROVIDER_CATEGORIES = [
  { id: 'custom', label: 'Custom (OpenAI / Anthropic Compatible)' },
  { id: 'oauth', label: 'OAuth Providers' },
  { id: 'free', label: 'Free Providers (No Key)' },
  { id: 'freeTier', label: 'Free Tier Providers' },
  { id: 'apikey', label: 'API Key Providers' },
  { id: 'webCookie', label: 'Web Cookie Providers' },
] as const

export const PROVIDER_CATALOG: ProviderCatalogItem[] = [
  {
    "id": "antigravity",
    "name": "Antigravity",
    "category": "oauth",
    "alias": "ag",
    "color": "#F59E0B",
    "icon": "rocket_launch",
    "noAuth": false
  },
  {
    "id": "claude",
    "name": "Claude Code",
    "category": "oauth",
    "alias": "cc",
    "color": "#D97757",
    "icon": "smart_toy",
    "noAuth": false
  },
  {
    "id": "cline",
    "name": "Cline",
    "category": "oauth",
    "alias": "cl",
    "color": "#5B9BD5",
    "icon": "smart_toy",
    "noAuth": false
  },
  {
    "id": "clinepass",
    "name": "ClinePass",
    "category": "oauth",
    "alias": "clinepass",
    "color": "#5B9BD5",
    "icon": "vpn_key",
    "noAuth": false
  },
  {
    "id": "codebuddy-intl",
    "name": "CodeBuddy",
    "category": "oauth",
    "alias": "cbai",
    "color": "#006EFF",
    "icon": "smart_toy",
    "noAuth": false
  },
  {
    "id": "codebuddy-cn",
    "name": "CodeBuddy CN",
    "category": "oauth",
    "alias": "cbcn",
    "color": "#006EFF",
    "icon": "smart_toy",
    "noAuth": false
  },
  {
    "id": "cursor",
    "name": "Cursor IDE",
    "category": "oauth",
    "alias": "cu",
    "color": "#00D4AA",
    "icon": "edit_note",
    "noAuth": false
  },
  {
    "id": "github",
    "name": "GitHub Copilot",
    "category": "oauth",
    "alias": "gh",
    "color": "#333333",
    "icon": "code",
    "noAuth": false
  },
  {
    "id": "gitlab",
    "name": "GitLab Duo",
    "category": "oauth",
    "alias": "gitlab",
    "color": "#FC6D26",
    "icon": "code",
    "noAuth": false
  },
  {
    "id": "grok-cli",
    "name": "Grok CLI (Grok Build)",
    "category": "oauth",
    "alias": "gcli",
    "color": "#1DA1F2",
    "icon": "auto_awesome",
    "noAuth": false
  },
  {
    "id": "iflow",
    "name": "iFlow AI",
    "category": "oauth",
    "alias": "if",
    "color": "#6366F1",
    "icon": "water_drop",
    "noAuth": false
  },
  {
    "id": "kilocode",
    "name": "Kilo Code",
    "category": "oauth",
    "alias": "kc",
    "color": "#FF6B35",
    "icon": "code",
    "noAuth": false
  },
  {
    "id": "kimi",
    "name": "Kimi",
    "category": "oauth",
    "alias": "kimi",
    "color": "#1E3A8A",
    "icon": "psychology",
    "noAuth": false
  },
  {
    "id": "codex",
    "name": "OpenAI Codex",
    "category": "oauth",
    "alias": "cx",
    "color": "#3B82F6",
    "icon": "code",
    "noAuth": false
  },
  {
    "id": "qoder",
    "name": "Qoder",
    "category": "oauth",
    "alias": "qd",
    "color": "#EC4899",
    "icon": "water_drop",
    "noAuth": false
  },
  {
    "id": "xai",
    "name": "xAI (Grok)",
    "category": "oauth",
    "alias": "xai",
    "color": "#1DA1F2",
    "icon": "auto_awesome",
    "noAuth": false
  },
  {
    "id": "xiaomi-mimo",
    "name": "Xiaomi MiMo",
    "category": "oauth",
    "alias": "mimo",
    "color": "#FF6900",
    "icon": "smart_toy",
    "noAuth": false
  },
  {
    "id": "zed",
    "name": "Zed",
    "category": "oauth",
    "alias": "zd",
    "color": "#A855F7",
    "icon": "code",
    "noAuth": false
  },
  {
    "id": "gemini-cli",
    "name": "Gemini CLI",
    "category": "free",
    "alias": "gc",
    "color": "#4285F4",
    "icon": "terminal",
    "noAuth": false
  },
  {
    "id": "kiro",
    "name": "Kiro AI",
    "category": "free",
    "alias": "kr",
    "color": "#FF6B35",
    "icon": "psychology_alt",
    "noAuth": false
  },
  {
    "id": "mimo-free",
    "name": "MiMo Code Free",
    "category": "free",
    "alias": "mmf",
    "color": "#FF6900",
    "icon": "smart_toy",
    "noAuth": true
  },
  {
    "id": "opencode",
    "name": "OpenCode Free",
    "category": "free",
    "alias": "oc",
    "color": "#E87040",
    "icon": "terminal",
    "noAuth": true
  },
  {
    "id": "api-airforce",
    "name": "API.airforce",
    "category": "freeTier",
    "alias": "af",
    "color": "#0EA5E9",
    "icon": "flight",
    "noAuth": false
  },
  {
    "id": "bazaarlink",
    "name": "Bazaarlink",
    "category": "freeTier",
    "alias": "bzl",
    "color": "#DC2626",
    "icon": "storefront",
    "noAuth": false
  },
  {
    "id": "byteplus",
    "name": "BytePlus ModelArk",
    "category": "freeTier",
    "alias": "bpm",
    "color": "#2563EB",
    "icon": "cloud",
    "noAuth": false
  },
  {
    "id": "cloudflare-ai",
    "name": "Cloudflare",
    "category": "freeTier",
    "alias": "cf",
    "color": "#F38020",
    "icon": "cloud",
    "noAuth": false
  },
  {
    "id": "coqui",
    "name": "Coqui TTS",
    "category": "freeTier",
    "alias": "coqui",
    "color": "#10B981",
    "icon": "record_voice_over",
    "noAuth": true
  },
  {
    "id": "edge-tts",
    "name": "Edge TTS",
    "category": "freeTier",
    "alias": "edge-tts",
    "color": "#0078D4",
    "icon": "record_voice_over",
    "noAuth": true
  },
  {
    "id": "gemini",
    "name": "Gemini",
    "category": "freeTier",
    "alias": "gemini",
    "color": "#4285F4",
    "icon": "diamond",
    "noAuth": false
  },
  {
    "id": "google-tts",
    "name": "Google TTS",
    "category": "freeTier",
    "alias": "google-tts",
    "color": "#4285F4",
    "icon": "record_voice_over",
    "noAuth": true
  },
  {
    "id": "kilo-gateway",
    "name": "Kilo Gateway",
    "category": "freeTier",
    "alias": "kgw",
    "color": "#8B5CF6",
    "icon": "login",
    "noAuth": false
  },
  {
    "id": "kimchi",
    "name": "Kimchi",
    "category": "freeTier",
    "alias": "kimchi",
    "color": "#FF521D",
    "icon": "restaurant",
    "noAuth": false
  },
  {
    "id": "local-device",
    "name": "Local Device",
    "category": "freeTier",
    "alias": "local-device",
    "color": "#64748B",
    "icon": "speaker",
    "noAuth": true
  },
  {
    "id": "nvidia",
    "name": "NVIDIA NIM",
    "category": "freeTier",
    "alias": "nvidia",
    "color": "#76B900",
    "icon": "developer_board",
    "noAuth": false
  },
  {
    "id": "ollama",
    "name": "Ollama Cloud",
    "category": "freeTier",
    "alias": "ollama",
    "color": "#ffffffff",
    "icon": "cloud",
    "noAuth": false
  },
  {
    "id": "openrouter",
    "name": "OpenRouter",
    "category": "freeTier",
    "alias": "openrouter",
    "color": "#F97316",
    "icon": "router",
    "noAuth": false
  },
  {
    "id": "poolside",
    "name": "Poolside",
    "category": "freeTier",
    "alias": "ps",
    "color": "#0EA5E9",
    "icon": "water_drop",
    "noAuth": false
  },
  {
    "id": "searxng",
    "name": "SearXNG",
    "category": "freeTier",
    "alias": "searxng",
    "color": "#3B82F6",
    "icon": "saved_search",
    "noAuth": true
  },
  {
    "id": "tortoise",
    "name": "Tortoise TTS",
    "category": "freeTier",
    "alias": "tortoise",
    "color": "#7C3AED",
    "icon": "record_voice_over",
    "noAuth": true
  },
  {
    "id": "vertex",
    "name": "Vertex AI",
    "category": "freeTier",
    "alias": "vx",
    "color": "#4285F4",
    "icon": "cloud",
    "noAuth": false
  },
  {
    "id": "alicode",
    "name": "Alibaba",
    "category": "apikey",
    "alias": "alicode",
    "color": "#FF6A00",
    "icon": "cloud",
    "noAuth": false
  },
  {
    "id": "alicode-intl",
    "name": "Alibaba Coding",
    "category": "apikey",
    "alias": "alicode-intl",
    "color": "#FF6A00",
    "icon": "cloud",
    "noAuth": false
  },
  {
    "id": "alims-intl",
    "name": "Alibaba Studio",
    "category": "apikey",
    "alias": "alims-intl",
    "color": "#FF6A00",
    "icon": "cloud",
    "noAuth": false
  },
  {
    "id": "alitp-intl",
    "name": "Alibaba Token Plan",
    "category": "apikey",
    "alias": "alitp-intl",
    "color": "#FF6A00",
    "icon": "cloud",
    "noAuth": false
  },
  {
    "id": "anthropic",
    "name": "Anthropic",
    "category": "apikey",
    "alias": "anthropic",
    "color": "#D97757",
    "icon": "smart_toy",
    "noAuth": false
  },
  {
    "id": "anthropic-version",
    "name": "anthropic-version",
    "category": "apikey",
    "alias": "anthropic-version",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false
  },
  {
    "id": "assemblyai",
    "name": "AssemblyAI",
    "category": "apikey",
    "alias": "aai",
    "color": "#0062FF",
    "icon": "record_voice_over",
    "noAuth": false
  },
  {
    "id": "aws-polly",
    "name": "AWS Polly",
    "category": "apikey",
    "alias": "polly",
    "color": "#FF9900",
    "icon": "record_voice_over",
    "noAuth": false
  },
  {
    "id": "azure",
    "name": "Azure OpenAI",
    "category": "apikey",
    "alias": "azure",
    "color": "#0078D4",
    "icon": "cloud",
    "noAuth": false
  },
  {
    "id": "baidu",
    "name": "Baidu Qianfan",
    "category": "apikey",
    "alias": "qianfan",
    "color": "#2932E1",
    "icon": "search",
    "noAuth": false
  },
  {
    "id": "black-forest-labs",
    "name": "Black Forest Labs",
    "category": "apikey",
    "alias": "bfl",
    "color": "#111827",
    "icon": "image",
    "noAuth": false
  },
  {
    "id": "blackbox",
    "name": "Blackbox AI",
    "category": "apikey",
    "alias": "bb",
    "color": "#5B5FEF",
    "icon": "smart_toy",
    "noAuth": false
  },
  {
    "id": "bluesminds",
    "name": "BluesMinds",
    "category": "apikey",
    "alias": "bm",
    "color": "#2563EB",
    "icon": "psychology",
    "noAuth": false
  },
  {
    "id": "brave-search",
    "name": "Brave Search",
    "category": "apikey",
    "alias": "brave",
    "color": "#FB542B",
    "icon": "travel_explore",
    "noAuth": false
  },
  {
    "id": "cartesia",
    "name": "Cartesia",
    "category": "apikey",
    "alias": "cartesia",
    "color": "#FF4F8B",
    "icon": "spatial_audio",
    "noAuth": false
  },
  {
    "id": "cerebras",
    "name": "Cerebras",
    "category": "apikey",
    "alias": "cerebras",
    "color": "#FF4F00",
    "icon": "memory",
    "noAuth": false
  },
  {
    "id": "chutes",
    "name": "Chutes AI",
    "category": "apikey",
    "alias": "ch",
    "color": "#ffffffff",
    "icon": "water_drop",
    "noAuth": false
  },
  {
    "id": "cohere",
    "name": "Cohere",
    "category": "apikey",
    "alias": "cohere",
    "color": "#39594D",
    "icon": "hub",
    "noAuth": false
  },
  {
    "id": "comfyui",
    "name": "ComfyUI",
    "category": "apikey",
    "alias": "comfyui",
    "color": "#4CAF50",
    "icon": "account_tree",
    "noAuth": false
  },
  {
    "id": "commandcode",
    "name": "Command Code",
    "category": "apikey",
    "alias": "cmc",
    "color": "#000000",
    "icon": "smart_toy",
    "noAuth": false
  },
  {
    "id": "content-type",
    "name": "content-type",
    "category": "apikey",
    "alias": "content-type",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false
  },
  {
    "id": "copilot-integration-id",
    "name": "copilot-integration-id",
    "category": "apikey",
    "alias": "copilot-integration-id",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false
  },
  {
    "id": "deepgram",
    "name": "Deepgram",
    "category": "apikey",
    "alias": "dg",
    "color": "#13EF93",
    "icon": "mic",
    "noAuth": false
  },
  {
    "id": "deepseek",
    "name": "DeepSeek",
    "category": "apikey",
    "alias": "ds",
    "color": "#4D6BFE",
    "icon": "bolt",
    "noAuth": false
  },
  {
    "id": "devin-cli",
    "name": "devin-cli",
    "category": "apikey",
    "alias": "devin-cli",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false
  },
  {
    "id": "editor-plugin-version",
    "name": "editor-plugin-version",
    "category": "apikey",
    "alias": "editor-plugin-version",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false
  },
  {
    "id": "editor-version",
    "name": "editor-version",
    "category": "apikey",
    "alias": "editor-version",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false
  },
  {
    "id": "elevenlabs",
    "name": "ElevenLabs",
    "category": "apikey",
    "alias": "el",
    "color": "#6C47FF",
    "icon": "record_voice_over",
    "noAuth": false
  },
  {
    "id": "exa",
    "name": "Exa",
    "category": "apikey",
    "alias": "exa",
    "color": "#2563EB",
    "icon": "manage_search",
    "noAuth": false
  },
  {
    "id": "fal-ai",
    "name": "Fal.ai",
    "category": "apikey",
    "alias": "fal",
    "color": "#2563EB",
    "icon": "image",
    "noAuth": false
  },
  {
    "id": "featherless",
    "name": "Featherless",
    "category": "apikey",
    "alias": "fl",
    "color": "#111827",
    "icon": "flutter_dash",
    "noAuth": false
  },
  {
    "id": "firecrawl",
    "name": "Firecrawl",
    "category": "apikey",
    "alias": "firecrawl",
    "color": "#F59E0B",
    "icon": "local_fire_department",
    "noAuth": false
  },
  {
    "id": "fireworks",
    "name": "Fireworks AI",
    "category": "apikey",
    "alias": "fireworks",
    "color": "#7B2EF2",
    "icon": "local_fire_department",
    "noAuth": false
  },
  {
    "id": "fish-audio",
    "name": "Fish Audio",
    "category": "apikey",
    "alias": "fish",
    "color": "#1E9BF0",
    "icon": "record_voice_over",
    "noAuth": false
  },
  {
    "id": "freebuff",
    "name": "freebuff",
    "category": "apikey",
    "alias": "freebuff",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false
  },
  {
    "id": "glm-cn",
    "name": "GLM (China)",
    "category": "apikey",
    "alias": "glm-cn",
    "color": "#DC2626",
    "icon": "code",
    "noAuth": false
  },
  {
    "id": "glm",
    "name": "GLM Coding",
    "category": "apikey",
    "alias": "glm",
    "color": "#2563EB",
    "icon": "code",
    "noAuth": false
  },
  {
    "id": "google-pse",
    "name": "Google PSE",
    "category": "apikey",
    "alias": "gpse",
    "color": "#4285F4",
    "icon": "search",
    "noAuth": false
  },
  {
    "id": "groq",
    "name": "Groq",
    "category": "apikey",
    "alias": "groq",
    "color": "#F55036",
    "icon": "speed",
    "noAuth": false
  },
  {
    "id": "huggingface",
    "name": "HuggingFace",
    "category": "apikey",
    "alias": "hf",
    "color": "#FFD21E",
    "icon": "face",
    "noAuth": false
  },
  {
    "id": "hyperbolic",
    "name": "Hyperbolic",
    "category": "apikey",
    "alias": "hyp",
    "color": "#00D4FF",
    "icon": "bolt",
    "noAuth": false
  },
  {
    "id": "inworld",
    "name": "Inworld TTS",
    "category": "apikey",
    "alias": "inworld",
    "color": "#FF6B6B",
    "icon": "record_voice_over",
    "noAuth": false
  },
  {
    "id": "jina-ai",
    "name": "Jina AI",
    "category": "apikey",
    "alias": "jina",
    "color": "#2563EB",
    "icon": "blur_on",
    "noAuth": false
  },
  {
    "id": "jina-reader",
    "name": "Jina Reader",
    "category": "apikey",
    "alias": "jina-reader",
    "color": "#000000",
    "icon": "menu_book",
    "noAuth": false
  },
  {
    "id": "kimi-coding",
    "name": "kimi-coding",
    "category": "apikey",
    "alias": "kimi-coding",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false
  },
  {
    "id": "linkup",
    "name": "Linkup",
    "category": "apikey",
    "alias": "linkup",
    "color": "#0EA5E9",
    "icon": "link",
    "noAuth": false
  },
  {
    "id": "llm7",
    "name": "LLM7",
    "category": "apikey",
    "alias": "llm7",
    "color": "#7C3AED",
    "icon": "pool",
    "noAuth": false
  },
  {
    "id": "minimax-cn",
    "name": "Minimax (China)",
    "category": "apikey",
    "alias": "minimax-cn",
    "color": "#DC2626",
    "icon": "memory",
    "noAuth": false
  },
  {
    "id": "minimax",
    "name": "Minimax Coding",
    "category": "apikey",
    "alias": "minimax",
    "color": "#7C3AED",
    "icon": "memory",
    "noAuth": false
  },
  {
    "id": "mistral",
    "name": "Mistral",
    "category": "apikey",
    "alias": "mistral",
    "color": "#FF7000",
    "icon": "air",
    "noAuth": false
  },
  {
    "id": "morph",
    "name": "Morph",
    "category": "apikey",
    "alias": "morph",
    "color": "#14B8A6",
    "icon": "change_history",
    "noAuth": false
  },
  {
    "id": "nanobanana",
    "name": "NanoBanana API",
    "category": "apikey",
    "alias": "nb",
    "color": "#FFD700",
    "icon": "extension",
    "noAuth": false
  },
  {
    "id": "nebius",
    "name": "Nebius AI",
    "category": "apikey",
    "alias": "nebius",
    "color": "#6C5CE7",
    "icon": "cloud",
    "noAuth": false
  },
  {
    "id": "ollama-local",
    "name": "Ollama Local",
    "category": "apikey",
    "alias": "ollama-local",
    "color": "#ffffffff",
    "icon": "cloud",
    "noAuth": false
  },
  {
    "id": "ollama-search",
    "name": "Ollama Search",
    "category": "apikey",
    "alias": "ollama-search",
    "color": "#ffffff",
    "icon": "cloud",
    "noAuth": false
  },
  {
    "id": "openai",
    "name": "OpenAI",
    "category": "apikey",
    "alias": "openai",
    "color": "#10A37F",
    "icon": "auto_awesome",
    "noAuth": false
  },
  {
    "id": "openai-intent",
    "name": "openai-intent",
    "category": "apikey",
    "alias": "openai-intent",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false
  },
  {
    "id": "opencode-go",
    "name": "OpenCode Go",
    "category": "apikey",
    "alias": "ocg",
    "color": "#E87040",
    "icon": "terminal",
    "noAuth": false
  },
  {
    "id": "originator",
    "name": "originator",
    "category": "apikey",
    "alias": "originator",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false
  },
  {
    "id": "perplexity",
    "name": "Perplexity",
    "category": "apikey",
    "alias": "pplx",
    "color": "#20808D",
    "icon": "search",
    "noAuth": false
  },
  {
    "id": "perplexity-agent",
    "name": "Perplexity Agent",
    "category": "apikey",
    "alias": "pa",
    "color": "#20808D",
    "icon": "travel_explore",
    "noAuth": false
  },
  {
    "id": "playht",
    "name": "PlayHT",
    "category": "apikey",
    "alias": "playht",
    "color": "#00B4D8",
    "icon": "play_circle",
    "noAuth": false
  },
  {
    "id": "recraft",
    "name": "Recraft",
    "category": "apikey",
    "alias": "recraft",
    "color": "#EC4899",
    "icon": "image",
    "noAuth": false
  },
  {
    "id": "runwayml",
    "name": "Runway ML",
    "category": "apikey",
    "alias": "runway",
    "color": "#000000",
    "icon": "movie",
    "noAuth": false
  },
  {
    "id": "sambanova",
    "name": "SambaNova",
    "category": "apikey",
    "alias": "samba",
    "color": "#F97316",
    "icon": "memory",
    "noAuth": false
  },
  {
    "id": "sdwebui",
    "name": "SD WebUI",
    "category": "apikey",
    "alias": "sdwebui",
    "color": "#FF7043",
    "icon": "brush",
    "noAuth": false
  },
  {
    "id": "searchapi",
    "name": "SearchAPI",
    "category": "apikey",
    "alias": "searchapi",
    "color": "#0EA5A4",
    "icon": "search",
    "noAuth": false
  },
  {
    "id": "serper",
    "name": "Serper",
    "category": "apikey",
    "alias": "serper",
    "color": "#4F46E5",
    "icon": "search",
    "noAuth": false
  },
  {
    "id": "siliconflow",
    "name": "SiliconFlow",
    "category": "apikey",
    "alias": "siliconflow",
    "color": "#5B6EF5",
    "icon": "cloud_queue",
    "noAuth": false
  },
  {
    "id": "stability-ai",
    "name": "Stability AI",
    "category": "apikey",
    "alias": "stability",
    "color": "#8B5CF6",
    "icon": "image",
    "noAuth": false
  },
  {
    "id": "tavily",
    "name": "Tavily",
    "category": "apikey",
    "alias": "tavily",
    "color": "#5B21B6",
    "icon": "search",
    "noAuth": false
  },
  {
    "id": "tencent",
    "name": "Tencent Hunyuan",
    "category": "apikey",
    "alias": "hunyuan",
    "color": "#0052D9",
    "icon": "cloud",
    "noAuth": false
  },
  {
    "id": "together",
    "name": "Together AI",
    "category": "apikey",
    "alias": "together",
    "color": "#0F6FFF",
    "icon": "group_work",
    "noAuth": false
  },
  {
    "id": "tokenrouter",
    "name": "TokenRouter",
    "category": "apikey",
    "alias": "tokenrouter",
    "color": "#0EA5E9",
    "icon": "hub",
    "noAuth": false
  },
  {
    "id": "topaz",
    "name": "Topaz",
    "category": "apikey",
    "alias": "topaz",
    "color": "#059669",
    "icon": "image",
    "noAuth": false
  },
  {
    "id": "trae",
    "name": "trae",
    "category": "apikey",
    "alias": "trae",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false
  },
  {
    "id": "user-agent",
    "name": "user-agent",
    "category": "apikey",
    "alias": "user-agent",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false
  },
  {
    "id": "venice",
    "name": "Venice AI",
    "category": "apikey",
    "alias": "venice",
    "color": "#DC2626",
    "icon": "shield",
    "noAuth": false
  },
  {
    "id": "vercel-ai-gateway",
    "name": "Vercel AI Gateway",
    "category": "apikey",
    "alias": "vercel",
    "color": "#111827",
    "icon": "deployed_code",
    "noAuth": false
  },
  {
    "id": "vertex-partner",
    "name": "Vertex Partner",
    "category": "apikey",
    "alias": "vxp",
    "color": "#34A853",
    "icon": "cloud",
    "noAuth": false
  },
  {
    "id": "volcengine-ark",
    "name": "Volcengine Ark",
    "category": "apikey",
    "alias": "ark",
    "color": "#1677FF",
    "icon": "cloud",
    "noAuth": false
  },
  {
    "id": "voyage-ai",
    "name": "Voyage AI",
    "category": "apikey",
    "alias": "voyage",
    "color": "#0EA5E9",
    "icon": "data_array",
    "noAuth": false
  },
  {
    "id": "windsurf",
    "name": "windsurf",
    "category": "apikey",
    "alias": "windsurf",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false
  },
  {
    "id": "x-codebuddy-request",
    "name": "x-codebuddy-request",
    "category": "apikey",
    "alias": "x-codebuddy-request",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false
  },
  {
    "id": "x-github-api-version",
    "name": "x-github-api-version",
    "category": "apikey",
    "alias": "x-github-api-version",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false
  },
  {
    "id": "x-requested-with",
    "name": "x-requested-with",
    "category": "apikey",
    "alias": "x-requested-with",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false
  },
  {
    "id": "x-vscode-user-agent-library-version",
    "name": "x-vscode-user-agent-library-version",
    "category": "apikey",
    "alias": "x-vscode-user-agent-library-version",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false
  },
  {
    "id": "xiaomi-tokenplan",
    "name": "Xiaomi MiMo (Token Plan)",
    "category": "apikey",
    "alias": "xmtp",
    "color": "#FF6700",
    "icon": "smart_toy",
    "noAuth": false
  },
  {
    "id": "xquik",
    "name": "Xquik",
    "category": "apikey",
    "alias": "xquik",
    "color": "#5C3327",
    "icon": "tag",
    "noAuth": false
  },
  {
    "id": "youcom",
    "name": "You.com Search",
    "category": "apikey",
    "alias": "youcom",
    "color": "#7C3AED",
    "icon": "search",
    "noAuth": false
  },
  {
    "id": "zai-search",
    "name": "zai-search",
    "category": "apikey",
    "alias": "zai-search",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false
  },
  {
    "id": "grok-web",
    "name": "Grok Web (Subscription)",
    "category": "webCookie",
    "alias": "gw",
    "color": "#1DA1F2",
    "icon": "auto_awesome",
    "noAuth": false
  },
  {
    "id": "perplexity-web",
    "name": "Perplexity Web (Pro/Max)",
    "category": "webCookie",
    "alias": "pw",
    "color": "#20808D",
    "icon": "search",
    "noAuth": false
  }
]

export const PROVIDER_CATALOG_MAP = new Map(PROVIDER_CATALOG.map((p) => [p.id, p]))
