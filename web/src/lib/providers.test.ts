import assert from 'node:assert'
import { describe, it } from 'node:test'
import {
  getProvidersByKind,
  isChatProvider,
  MEDIA_PROVIDER_KINDS,
  PROVIDER_CATALOG,
} from './providers'
import { pathToTab, TAB_ROUTES } from './router'

describe('providers & media separation', () => {
  it('correctly classifies chat vs pure media providers', () => {
    const openai = PROVIDER_CATALOG.find((p) => p.id === 'openai')
    const anthropic = PROVIDER_CATALOG.find((p) => p.id === 'anthropic')
    const elevenlabs = PROVIDER_CATALOG.find((p) => p.id === 'elevenlabs')
    const deepgram = PROVIDER_CATALOG.find((p) => p.id === 'deepgram')
    const bfl = PROVIDER_CATALOG.find((p) => p.id === 'black-forest-labs')
    const voyage = PROVIDER_CATALOG.find((p) => p.id === 'voyage-ai')
    const runway = PROVIDER_CATALOG.find((p) => p.id === 'runwayml')
    const brave = PROVIDER_CATALOG.find((p) => p.id === 'brave-search')

    assert.ok(openai && isChatProvider(openai))
    assert.ok(anthropic && isChatProvider(anthropic))
    assert.ok(elevenlabs && !isChatProvider(elevenlabs))
    assert.ok(deepgram && !isChatProvider(deepgram))
    assert.ok(bfl && !isChatProvider(bfl))
    assert.ok(voyage && !isChatProvider(voyage))
    assert.ok(runway && !isChatProvider(runway))
    assert.ok(brave && !isChatProvider(brave))
  })

  it('retrieves providers by kind', () => {
    const ttsProviders = getProvidersByKind('tts')
    assert.ok(ttsProviders.some((p) => p.id === 'elevenlabs'))
    assert.ok(ttsProviders.some((p) => p.id === 'coqui'))
    assert.ok(ttsProviders.some((p) => p.id === 'openai'))
    assert.strictEqual(ttsProviders.some((p) => p.id === 'anthropic'), false)

    const sttProviders = getProvidersByKind('stt')
    assert.ok(sttProviders.some((p) => p.id === 'assemblyai'))
    assert.ok(sttProviders.some((p) => p.id === 'deepgram'))
    assert.ok(sttProviders.some((p) => p.id === 'openai'))

    const imageProviders = getProvidersByKind('image')
    assert.ok(imageProviders.some((p) => p.id === 'black-forest-labs'))
    assert.ok(imageProviders.some((p) => p.id === 'fal-ai'))
    assert.ok(imageProviders.some((p) => p.id === 'runwayml'))

    const videoProviders = getProvidersByKind('video')
    assert.ok(videoProviders.some((p) => p.id === 'runwayml'))

    const embeddingProviders = getProvidersByKind('embedding')
    assert.ok(embeddingProviders.some((p) => p.id === 'voyage-ai'))
    assert.ok(embeddingProviders.some((p) => p.id === 'openai'))

    assert.ok(MEDIA_PROVIDER_KINDS.length >= 6)
  })

  it('maps media router routes correctly', () => {
    assert.strictEqual(pathToTab('/dashboard/media-providers/embedding'), 'media-embedding')
    assert.strictEqual(pathToTab('/dashboard/media-providers/image'), 'media-image')
    assert.strictEqual(pathToTab('/dashboard/media-providers/tts'), 'media-tts')
    assert.strictEqual(pathToTab('/dashboard/media-providers/stt'), 'media-stt')
    assert.strictEqual(pathToTab('/dashboard/media-providers/video'), 'media-video')
    assert.strictEqual(pathToTab('/dashboard/media-providers/web'), 'media-web')

    assert.strictEqual(TAB_ROUTES['media-embedding'], '/dashboard/media-providers/embedding')
    assert.strictEqual(TAB_ROUTES['media-image'], '/dashboard/media-providers/image')
    assert.strictEqual(TAB_ROUTES['media-tts'], '/dashboard/media-providers/tts')
    assert.strictEqual(TAB_ROUTES['media-stt'], '/dashboard/media-providers/stt')
    assert.strictEqual(TAB_ROUTES['media-video'], '/dashboard/media-providers/video')
    assert.strictEqual(TAB_ROUTES['media-web'], '/dashboard/media-providers/web')
  })
})
