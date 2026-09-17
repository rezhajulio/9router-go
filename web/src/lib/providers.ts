// Auto-generated provider catalog matching upstream 9router
export interface ProviderCatalogItem {
  id: string
  name: string
  category: 'oauth' | 'free' | 'freeTier' | 'apikey' | 'webCookie' | 'custom'
  alias: string
  color: string
  icon: string
  noAuth?: boolean
  serviceKinds?: string[]
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
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "claude",
    "name": "Claude Code",
    "category": "oauth",
    "alias": "cc",
    "color": "#D97757",
    "icon": "smart_toy",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "cline",
    "name": "Cline",
    "category": "oauth",
    "alias": "cl",
    "color": "#5B9BD5",
    "icon": "smart_toy",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "clinepass",
    "name": "ClinePass",
    "category": "oauth",
    "alias": "clinepass",
    "color": "#5B9BD5",
    "icon": "vpn_key",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "codebuddy-intl",
    "name": "CodeBuddy",
    "category": "oauth",
    "alias": "cbai",
    "color": "#006EFF",
    "icon": "smart_toy",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "codebuddy-cn",
    "name": "CodeBuddy CN",
    "category": "oauth",
    "alias": "cbcn",
    "color": "#006EFF",
    "icon": "smart_toy",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "cursor",
    "name": "Cursor IDE",
    "category": "oauth",
    "alias": "cu",
    "color": "#00D4AA",
    "icon": "edit_note",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "github",
    "name": "GitHub Copilot",
    "category": "oauth",
    "alias": "gh",
    "color": "#333333",
    "icon": "code",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "gitlab",
    "name": "GitLab Duo",
    "category": "oauth",
    "alias": "gitlab",
    "color": "#FC6D26",
    "icon": "code",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "grok-cli",
    "name": "Grok CLI (Grok Build)",
    "category": "oauth",
    "alias": "gcli",
    "color": "#1DA1F2",
    "icon": "auto_awesome",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "iflow",
    "name": "iFlow AI",
    "category": "oauth",
    "alias": "if",
    "color": "#6366F1",
    "icon": "water_drop",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "kilocode",
    "name": "Kilo Code",
    "category": "oauth",
    "alias": "kc",
    "color": "#FF6B35",
    "icon": "code",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "kimi",
    "name": "Kimi",
    "category": "oauth",
    "alias": "kimi",
    "color": "#1E3A8A",
    "icon": "psychology",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "codex",
    "name": "OpenAI Codex",
    "category": "oauth",
    "alias": "cx",
    "color": "#3B82F6",
    "icon": "code",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "qoder",
    "name": "Qoder",
    "category": "oauth",
    "alias": "qd",
    "color": "#EC4899",
    "icon": "water_drop",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "xai",
    "name": "xAI (Grok)",
    "category": "oauth",
    "alias": "xai",
    "color": "#1DA1F2",
    "icon": "auto_awesome",
    "noAuth": false,
    "serviceKinds": [
      "llm",
      "image",
      "video"
    ]
  },
  {
    "id": "xiaomi-mimo",
    "name": "Xiaomi MiMo",
    "category": "oauth",
    "alias": "mimo",
    "color": "#FF6900",
    "icon": "smart_toy",
    "noAuth": false,
    "serviceKinds": [
      "llm",
      "tts"
    ]
  },
  {
    "id": "zed",
    "name": "Zed",
    "category": "oauth",
    "alias": "zd",
    "color": "#A855F7",
    "icon": "code",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "freebuff",
    "name": "Freebuff",
    "category": "free",
    "alias": "fb",
    "color": "#0a0a0b",
    "icon": "bolt",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "gemini-cli",
    "name": "Gemini CLI",
    "category": "free",
    "alias": "gc",
    "color": "#4285F4",
    "icon": "terminal",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "kiro",
    "name": "Kiro AI",
    "category": "free",
    "alias": "kr",
    "color": "#FF6B35",
    "icon": "psychology_alt",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "mimo-free",
    "name": "MiMo Code Free",
    "category": "free",
    "alias": "mmf",
    "color": "#FF6900",
    "icon": "smart_toy",
    "noAuth": true,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "opencode",
    "name": "OpenCode Free",
    "category": "free",
    "alias": "oc",
    "color": "#E87040",
    "icon": "terminal",
    "noAuth": true,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "api-airforce",
    "name": "API.airforce",
    "category": "freeTier",
    "alias": "af",
    "color": "#0EA5E9",
    "icon": "flight",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "bazaarlink",
    "name": "Bazaarlink",
    "category": "freeTier",
    "alias": "bzl",
    "color": "#DC2626",
    "icon": "storefront",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "byteplus",
    "name": "BytePlus ModelArk",
    "category": "freeTier",
    "alias": "bpm",
    "color": "#2563EB",
    "icon": "cloud",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "cloudflare-ai",
    "name": "Cloudflare",
    "category": "freeTier",
    "alias": "cf",
    "color": "#F38020",
    "icon": "cloud",
    "noAuth": false,
    "serviceKinds": [
      "llm",
      "image"
    ]
  },
  {
    "id": "coqui",
    "name": "Coqui TTS",
    "category": "freeTier",
    "alias": "coqui",
    "color": "#10B981",
    "icon": "record_voice_over",
    "noAuth": true,
    "serviceKinds": [
      "tts"
    ]
  },
  {
    "id": "edge-tts",
    "name": "Edge TTS",
    "category": "freeTier",
    "alias": "edge-tts",
    "color": "#0078D4",
    "icon": "record_voice_over",
    "noAuth": true,
    "serviceKinds": [
      "tts"
    ]
  },
  {
    "id": "gemini",
    "name": "Gemini",
    "category": "freeTier",
    "alias": "gemini",
    "color": "#4285F4",
    "icon": "diamond",
    "noAuth": false,
    "serviceKinds": [
      "llm",
      "image",
      "tts",
      "stt",
      "embedding"
    ]
  },
  {
    "id": "google-tts",
    "name": "Google TTS",
    "category": "freeTier",
    "alias": "google-tts",
    "color": "#4285F4",
    "icon": "record_voice_over",
    "noAuth": true,
    "serviceKinds": [
      "tts"
    ]
  },
  {
    "id": "kilo-gateway",
    "name": "Kilo Gateway",
    "category": "freeTier",
    "alias": "kgw",
    "color": "#8B5CF6",
    "icon": "login",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "kimchi",
    "name": "Kimchi",
    "category": "freeTier",
    "alias": "kimchi",
    "color": "#FF521D",
    "icon": "restaurant",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "local-device",
    "name": "Local Device",
    "category": "freeTier",
    "alias": "local-device",
    "color": "#64748B",
    "icon": "speaker",
    "noAuth": true,
    "serviceKinds": [
      "tts"
    ]
  },
  {
    "id": "nvidia",
    "name": "NVIDIA NIM",
    "category": "freeTier",
    "alias": "nvidia",
    "color": "#76B900",
    "icon": "developer_board",
    "noAuth": false,
    "serviceKinds": [
      "llm",
      "stt",
      "tts",
      "embedding"
    ]
  },
  {
    "id": "ollama",
    "name": "Ollama Cloud",
    "category": "freeTier",
    "alias": "ollama",
    "color": "#ffffffff",
    "icon": "cloud",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "openrouter",
    "name": "OpenRouter",
    "category": "freeTier",
    "alias": "openrouter",
    "color": "#F97316",
    "icon": "router",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "poolside",
    "name": "Poolside",
    "category": "freeTier",
    "alias": "ps",
    "color": "#0EA5E9",
    "icon": "water_drop",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "searxng",
    "name": "SearXNG",
    "category": "freeTier",
    "alias": "searxng",
    "color": "#3B82F6",
    "icon": "saved_search",
    "noAuth": true,
    "serviceKinds": [
      "webSearch",
      "web"
    ]
  },
  {
    "id": "tortoise",
    "name": "Tortoise TTS",
    "category": "freeTier",
    "alias": "tortoise",
    "color": "#7C3AED",
    "icon": "record_voice_over",
    "noAuth": true,
    "serviceKinds": [
      "tts"
    ]
  },
  {
    "id": "vertex",
    "name": "Vertex AI",
    "category": "freeTier",
    "alias": "vx",
    "color": "#4285F4",
    "icon": "cloud",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "alicode",
    "name": "Alibaba",
    "category": "apikey",
    "alias": "alicode",
    "color": "#FF6A00",
    "icon": "cloud",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "alicode-intl",
    "name": "Alibaba Coding",
    "category": "apikey",
    "alias": "alicode-intl",
    "color": "#FF6A00",
    "icon": "cloud",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "alims-intl",
    "name": "Alibaba Studio",
    "category": "apikey",
    "alias": "alims-intl",
    "color": "#FF6A00",
    "icon": "cloud",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "alitp-intl",
    "name": "Alibaba Token Plan",
    "category": "apikey",
    "alias": "alitp-intl",
    "color": "#FF6A00",
    "icon": "cloud",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "anthropic",
    "name": "Anthropic",
    "category": "apikey",
    "alias": "anthropic",
    "color": "#D97757",
    "icon": "smart_toy",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "anthropic-version",
    "name": "anthropic-version",
    "category": "apikey",
    "alias": "anthropic-version",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "assemblyai",
    "name": "AssemblyAI",
    "category": "apikey",
    "alias": "aai",
    "color": "#0062FF",
    "icon": "record_voice_over",
    "noAuth": false,
    "serviceKinds": [
      "stt"
    ]
  },
  {
    "id": "aws-polly",
    "name": "AWS Polly",
    "category": "apikey",
    "alias": "polly",
    "color": "#FF9900",
    "icon": "record_voice_over",
    "noAuth": false,
    "serviceKinds": [
      "tts"
    ]
  },
  {
    "id": "azure",
    "name": "Azure OpenAI",
    "category": "apikey",
    "alias": "azure",
    "color": "#0078D4",
    "icon": "cloud",
    "noAuth": false,
    "serviceKinds": [
      "llm",
      "image",
      "tts",
      "stt",
      "embedding"
    ]
  },
  {
    "id": "baidu",
    "name": "Baidu Qianfan",
    "category": "apikey",
    "alias": "qianfan",
    "color": "#2932E1",
    "icon": "search",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "black-forest-labs",
    "name": "Black Forest Labs",
    "category": "apikey",
    "alias": "bfl",
    "color": "#111827",
    "icon": "image",
    "noAuth": false,
    "serviceKinds": [
      "image"
    ]
  },
  {
    "id": "blackbox",
    "name": "Blackbox AI",
    "category": "apikey",
    "alias": "bb",
    "color": "#5B5FEF",
    "icon": "smart_toy",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "bluesminds",
    "name": "BluesMinds",
    "category": "apikey",
    "alias": "bm",
    "color": "#2563EB",
    "icon": "psychology",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "brave-search",
    "name": "Brave Search",
    "category": "apikey",
    "alias": "brave",
    "color": "#FB542B",
    "icon": "travel_explore",
    "noAuth": false,
    "serviceKinds": [
      "webSearch",
      "web"
    ]
  },
  {
    "id": "cartesia",
    "name": "Cartesia",
    "category": "apikey",
    "alias": "cartesia",
    "color": "#FF4F8B",
    "icon": "spatial_audio",
    "noAuth": false,
    "serviceKinds": [
      "tts"
    ]
  },
  {
    "id": "cerebras",
    "name": "Cerebras",
    "category": "apikey",
    "alias": "cerebras",
    "color": "#FF4F00",
    "icon": "memory",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "chutes",
    "name": "Chutes AI",
    "category": "apikey",
    "alias": "ch",
    "color": "#ffffffff",
    "icon": "water_drop",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "cohere",
    "name": "Cohere",
    "category": "apikey",
    "alias": "cohere",
    "color": "#39594D",
    "icon": "hub",
    "noAuth": false,
    "serviceKinds": [
      "llm",
      "embedding"
    ]
  },
  {
    "id": "comfyui",
    "name": "ComfyUI",
    "category": "apikey",
    "alias": "comfyui",
    "color": "#4CAF50",
    "icon": "account_tree",
    "noAuth": false,
    "serviceKinds": [
      "image"
    ]
  },
  {
    "id": "commandcode",
    "name": "Command Code",
    "category": "apikey",
    "alias": "cmc",
    "color": "#000000",
    "icon": "smart_toy",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "content-type",
    "name": "content-type",
    "category": "apikey",
    "alias": "content-type",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "copilot-integration-id",
    "name": "copilot-integration-id",
    "category": "apikey",
    "alias": "copilot-integration-id",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "deepgram",
    "name": "Deepgram",
    "category": "apikey",
    "alias": "dg",
    "color": "#13EF93",
    "icon": "mic",
    "noAuth": false,
    "serviceKinds": [
      "stt"
    ]
  },
  {
    "id": "deepseek",
    "name": "DeepSeek",
    "category": "apikey",
    "alias": "ds",
    "color": "#4D6BFE",
    "icon": "bolt",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "devin-cli",
    "name": "devin-cli",
    "category": "apikey",
    "alias": "devin-cli",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "editor-plugin-version",
    "name": "editor-plugin-version",
    "category": "apikey",
    "alias": "editor-plugin-version",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "editor-version",
    "name": "editor-version",
    "category": "apikey",
    "alias": "editor-version",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "elevenlabs",
    "name": "ElevenLabs",
    "category": "apikey",
    "alias": "el",
    "color": "#6C47FF",
    "icon": "record_voice_over",
    "noAuth": false,
    "serviceKinds": [
      "tts"
    ]
  },
  {
    "id": "exa",
    "name": "Exa",
    "category": "apikey",
    "alias": "exa",
    "color": "#2563EB",
    "icon": "manage_search",
    "noAuth": false,
    "serviceKinds": [
      "webSearch",
      "web"
    ]
  },
  {
    "id": "fal-ai",
    "name": "Fal.ai",
    "category": "apikey",
    "alias": "fal",
    "color": "#2563EB",
    "icon": "image",
    "noAuth": false,
    "serviceKinds": [
      "image"
    ]
  },
  {
    "id": "featherless",
    "name": "Featherless",
    "category": "apikey",
    "alias": "fl",
    "color": "#111827",
    "icon": "flutter_dash",
    "noAuth": false,
    "serviceKinds": [
      "llm",
      "embedding"
    ]
  },
  {
    "id": "firecrawl",
    "name": "Firecrawl",
    "category": "apikey",
    "alias": "firecrawl",
    "color": "#F59E0B",
    "icon": "local_fire_department",
    "noAuth": false,
    "serviceKinds": [
      "webFetch",
      "web"
    ]
  },
  {
    "id": "fireworks",
    "name": "Fireworks AI",
    "category": "apikey",
    "alias": "fireworks",
    "color": "#7B2EF2",
    "icon": "local_fire_department",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "fish-audio",
    "name": "Fish Audio",
    "category": "apikey",
    "alias": "fish",
    "color": "#1E9BF0",
    "icon": "record_voice_over",
    "noAuth": false,
    "serviceKinds": [
      "tts"
    ]
  },
  {
    "id": "glm-cn",
    "name": "GLM (China)",
    "category": "apikey",
    "alias": "glm-cn",
    "color": "#DC2626",
    "icon": "code",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "glm",
    "name": "GLM Coding",
    "category": "apikey",
    "alias": "glm",
    "color": "#2563EB",
    "icon": "code",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "google-pse",
    "name": "Google PSE",
    "category": "apikey",
    "alias": "gpse",
    "color": "#4285F4",
    "icon": "search",
    "noAuth": false,
    "serviceKinds": [
      "webSearch",
      "web"
    ]
  },
  {
    "id": "groq",
    "name": "Groq",
    "category": "apikey",
    "alias": "groq",
    "color": "#F55036",
    "icon": "speed",
    "noAuth": false,
    "serviceKinds": [
      "llm",
      "stt"
    ]
  },
  {
    "id": "huggingface",
    "name": "HuggingFace",
    "category": "apikey",
    "alias": "hf",
    "color": "#FFD21E",
    "icon": "face",
    "noAuth": false,
    "serviceKinds": [
      "llm",
      "image",
      "stt"
    ]
  },
  {
    "id": "hyperbolic",
    "name": "Hyperbolic",
    "category": "apikey",
    "alias": "hyp",
    "color": "#00D4FF",
    "icon": "bolt",
    "noAuth": false,
    "serviceKinds": [
      "llm",
      "tts"
    ]
  },
  {
    "id": "inworld",
    "name": "Inworld TTS",
    "category": "apikey",
    "alias": "inworld",
    "color": "#FF6B6B",
    "icon": "record_voice_over",
    "noAuth": false,
    "serviceKinds": [
      "tts"
    ]
  },
  {
    "id": "jina-ai",
    "name": "Jina AI",
    "category": "apikey",
    "alias": "jina",
    "color": "#2563EB",
    "icon": "blur_on",
    "noAuth": false,
    "serviceKinds": [
      "webSearch",
      "web"
    ]
  },
  {
    "id": "jina-reader",
    "name": "Jina Reader",
    "category": "apikey",
    "alias": "jina-reader",
    "color": "#000000",
    "icon": "menu_book",
    "noAuth": false,
    "serviceKinds": [
      "webFetch",
      "web"
    ]
  },
  {
    "id": "kimi-coding",
    "name": "kimi-coding",
    "category": "apikey",
    "alias": "kimi-coding",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "linkup",
    "name": "Linkup",
    "category": "apikey",
    "alias": "linkup",
    "color": "#0EA5E9",
    "icon": "link",
    "noAuth": false,
    "serviceKinds": [
      "webSearch",
      "web"
    ]
  },
  {
    "id": "llm7",
    "name": "LLM7",
    "category": "apikey",
    "alias": "llm7",
    "color": "#7C3AED",
    "icon": "pool",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "minimax-cn",
    "name": "Minimax (China)",
    "category": "apikey",
    "alias": "minimax-cn",
    "color": "#DC2626",
    "icon": "memory",
    "noAuth": false,
    "serviceKinds": [
      "llm",
      "image",
      "tts"
    ]
  },
  {
    "id": "minimax",
    "name": "Minimax Coding",
    "category": "apikey",
    "alias": "minimax",
    "color": "#7C3AED",
    "icon": "memory",
    "noAuth": false,
    "serviceKinds": [
      "llm",
      "image",
      "tts"
    ]
  },
  {
    "id": "mistral",
    "name": "Mistral",
    "category": "apikey",
    "alias": "mistral",
    "color": "#FF7000",
    "icon": "air",
    "noAuth": false,
    "serviceKinds": [
      "llm",
      "embedding"
    ]
  },
  {
    "id": "morph",
    "name": "Morph",
    "category": "apikey",
    "alias": "morph",
    "color": "#14B8A6",
    "icon": "change_history",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "nanobanana",
    "name": "NanoBanana API",
    "category": "apikey",
    "alias": "nb",
    "color": "#FFD700",
    "icon": "extension",
    "noAuth": false,
    "serviceKinds": [
      "image"
    ]
  },
  {
    "id": "nebius",
    "name": "Nebius AI",
    "category": "apikey",
    "alias": "nebius",
    "color": "#6C5CE7",
    "icon": "cloud",
    "noAuth": false,
    "serviceKinds": [
      "llm",
      "embedding"
    ]
  },
  {
    "id": "ollama-local",
    "name": "Ollama Local",
    "category": "apikey",
    "alias": "ollama-local",
    "color": "#ffffffff",
    "icon": "cloud",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "ollama-search",
    "name": "Ollama Search",
    "category": "apikey",
    "alias": "ollama-search",
    "color": "#ffffff",
    "icon": "cloud",
    "noAuth": false,
    "serviceKinds": [
      "webSearch",
      "web"
    ]
  },
  {
    "id": "openai",
    "name": "OpenAI",
    "category": "apikey",
    "alias": "openai",
    "color": "#10A37F",
    "icon": "auto_awesome",
    "noAuth": false,
    "serviceKinds": [
      "llm",
      "image",
      "tts",
      "stt",
      "embedding"
    ]
  },
  {
    "id": "openai-intent",
    "name": "openai-intent",
    "category": "apikey",
    "alias": "openai-intent",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "opencode-go",
    "name": "OpenCode Go",
    "category": "apikey",
    "alias": "ocg",
    "color": "#E87040",
    "icon": "terminal",
    "noAuth": false,
    "serviceKinds": [
      "llm",
      "image",
      "video",
      "tts",
      "embedding"
    ]
  },
  {
    "id": "originator",
    "name": "originator",
    "category": "apikey",
    "alias": "originator",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "perplexity",
    "name": "Perplexity",
    "category": "apikey",
    "alias": "pplx",
    "color": "#20808D",
    "icon": "search",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "perplexity-agent",
    "name": "Perplexity Agent",
    "category": "apikey",
    "alias": "pa",
    "color": "#20808D",
    "icon": "travel_explore",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "playht",
    "name": "PlayHT",
    "category": "apikey",
    "alias": "playht",
    "color": "#00B4D8",
    "icon": "play_circle",
    "noAuth": false,
    "serviceKinds": [
      "tts"
    ]
  },
  {
    "id": "recraft",
    "name": "Recraft",
    "category": "apikey",
    "alias": "recraft",
    "color": "#EC4899",
    "icon": "image",
    "noAuth": false,
    "serviceKinds": [
      "image"
    ]
  },
  {
    "id": "runwayml",
    "name": "Runway ML",
    "category": "apikey",
    "alias": "runway",
    "color": "#000000",
    "icon": "movie",
    "noAuth": false,
    "serviceKinds": [
      "image",
      "video"
    ]
  },
  {
    "id": "sambanova",
    "name": "SambaNova",
    "category": "apikey",
    "alias": "samba",
    "color": "#F97316",
    "icon": "memory",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "sdwebui",
    "name": "SD WebUI",
    "category": "apikey",
    "alias": "sdwebui",
    "color": "#FF7043",
    "icon": "brush",
    "noAuth": false,
    "serviceKinds": [
      "image"
    ]
  },
  {
    "id": "searchapi",
    "name": "SearchAPI",
    "category": "apikey",
    "alias": "searchapi",
    "color": "#0EA5A4",
    "icon": "search",
    "noAuth": false,
    "serviceKinds": [
      "webSearch",
      "web"
    ]
  },
  {
    "id": "serper",
    "name": "Serper",
    "category": "apikey",
    "alias": "serper",
    "color": "#4F46E5",
    "icon": "search",
    "noAuth": false,
    "serviceKinds": [
      "webSearch",
      "web"
    ]
  },
  {
    "id": "siliconflow",
    "name": "SiliconFlow",
    "category": "apikey",
    "alias": "siliconflow",
    "color": "#5B6EF5",
    "icon": "cloud_queue",
    "noAuth": false,
    "serviceKinds": [
      "llm",
      "image"
    ]
  },
  {
    "id": "stability-ai",
    "name": "Stability AI",
    "category": "apikey",
    "alias": "stability",
    "color": "#8B5CF6",
    "icon": "image",
    "noAuth": false,
    "serviceKinds": [
      "image"
    ]
  },
  {
    "id": "tavily",
    "name": "Tavily",
    "category": "apikey",
    "alias": "tavily",
    "color": "#5B21B6",
    "icon": "search",
    "noAuth": false,
    "serviceKinds": [
      "webSearch",
      "web"
    ]
  },
  {
    "id": "tencent",
    "name": "Tencent Hunyuan",
    "category": "apikey",
    "alias": "hunyuan",
    "color": "#0052D9",
    "icon": "cloud",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "together",
    "name": "Together AI",
    "category": "apikey",
    "alias": "together",
    "color": "#0F6FFF",
    "icon": "group_work",
    "noAuth": false,
    "serviceKinds": [
      "llm",
      "image",
      "embedding"
    ]
  },
  {
    "id": "tokenrouter",
    "name": "TokenRouter",
    "category": "apikey",
    "alias": "tokenrouter",
    "color": "#0EA5E9",
    "icon": "hub",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "topaz",
    "name": "Topaz",
    "category": "apikey",
    "alias": "topaz",
    "color": "#059669",
    "icon": "image",
    "noAuth": false,
    "serviceKinds": [
      "image"
    ]
  },
  {
    "id": "trae",
    "name": "trae",
    "category": "apikey",
    "alias": "trae",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "user-agent",
    "name": "user-agent",
    "category": "apikey",
    "alias": "user-agent",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "venice",
    "name": "Venice AI",
    "category": "apikey",
    "alias": "venice",
    "color": "#DC2626",
    "icon": "shield",
    "noAuth": false,
    "serviceKinds": [
      "llm",
      "image",
      "embedding"
    ]
  },
  {
    "id": "vercel-ai-gateway",
    "name": "Vercel AI Gateway",
    "category": "apikey",
    "alias": "vercel",
    "color": "#111827",
    "icon": "deployed_code",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "vertex-partner",
    "name": "Vertex Partner",
    "category": "apikey",
    "alias": "vxp",
    "color": "#34A853",
    "icon": "cloud",
    "noAuth": false,
    "serviceKinds": [
      "llm",
      "video"
    ]
  },
  {
    "id": "volcengine-ark",
    "name": "Volcengine Ark",
    "category": "apikey",
    "alias": "ark",
    "color": "#1677FF",
    "icon": "cloud",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "voyage-ai",
    "name": "Voyage AI",
    "category": "apikey",
    "alias": "voyage",
    "color": "#0EA5E9",
    "icon": "data_array",
    "noAuth": false,
    "serviceKinds": [
      "embedding"
    ]
  },
  {
    "id": "windsurf",
    "name": "windsurf",
    "category": "apikey",
    "alias": "windsurf",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "x-codebuddy-request",
    "name": "x-codebuddy-request",
    "category": "apikey",
    "alias": "x-codebuddy-request",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "x-github-api-version",
    "name": "x-github-api-version",
    "category": "apikey",
    "alias": "x-github-api-version",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "x-requested-with",
    "name": "x-requested-with",
    "category": "apikey",
    "alias": "x-requested-with",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "x-vscode-user-agent-library-version",
    "name": "x-vscode-user-agent-library-version",
    "category": "apikey",
    "alias": "x-vscode-user-agent-library-version",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "xiaomi-tokenplan",
    "name": "Xiaomi MiMo (Token Plan)",
    "category": "apikey",
    "alias": "xmtp",
    "color": "#FF6700",
    "icon": "smart_toy",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "xquik",
    "name": "Xquik",
    "category": "apikey",
    "alias": "xquik",
    "color": "#5C3327",
    "icon": "tag",
    "noAuth": false,
    "serviceKinds": [
      "webSearch",
      "web"
    ]
  },
  {
    "id": "youcom",
    "name": "You.com Search",
    "category": "apikey",
    "alias": "youcom",
    "color": "#7C3AED",
    "icon": "search",
    "noAuth": false,
    "serviceKinds": [
      "webSearch",
      "web"
    ]
  },
  {
    "id": "zai-search",
    "name": "zai-search",
    "category": "apikey",
    "alias": "zai-search",
    "color": "#888888",
    "icon": "dns",
    "noAuth": false,
    "serviceKinds": [
      "webSearch",
      "web"
    ]
  },
  {
    "id": "grok-web",
    "name": "Grok Web (Subscription)",
    "category": "webCookie",
    "alias": "gw",
    "color": "#1DA1F2",
    "icon": "auto_awesome",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "perplexity-web",
    "name": "Perplexity Web (Pro/Max)",
    "category": "webCookie",
    "alias": "pw",
    "color": "#20808D",
    "icon": "search",
    "noAuth": false,
    "serviceKinds": [
      "llm"
    ]
  },
  {
    "id": "selfhosted-tts",
    "name": "Self-Hosted TTS",
    "category": "apikey",
    "alias": "selfhosted-tts",
    "color": "#10B981",
    "icon": "volume_up",
    "noAuth": true,
    "serviceKinds": [
      "tts"
    ]
  },
  {
    "id": "selfhosted-stt",
    "name": "Self-Hosted STT",
    "category": "apikey",
    "alias": "selfhosted-stt",
    "color": "#3B82F6",
    "icon": "mic",
    "noAuth": true,
    "serviceKinds": [
      "stt"
    ]
  },
  {
    "id": "selfhosted-embedding",
    "name": "Self-Hosted Embedding",
    "category": "apikey",
    "alias": "selfhosted-embedding",
    "color": "#8B5CF6",
    "icon": "layers",
    "noAuth": true,
    "serviceKinds": [
      "embedding"
    ]
  }
]

export const PROVIDER_CATALOG_MAP = new Map(PROVIDER_CATALOG.map((p) => [p.id, p]))

export function isChatProvider(p: ProviderCatalogItem): boolean {
  return (p.serviceKinds ?? ['llm']).includes('llm')
}

export function getProvidersByKind(kind: string): ProviderCatalogItem[] {
  return PROVIDER_CATALOG.filter((p) => (p.serviceKinds ?? ['llm']).includes(kind))
}

export const MEDIA_PROVIDER_KINDS = [
  'embedding',
  'image',
  'tts',
  'stt',
  'video',
  'webSearch',
  'webFetch',
  'web',
] as const
