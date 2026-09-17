import assert from 'node:assert'
import { describe, it } from 'node:test'
import type { Combo, ProviderConnection, ProviderNode } from '../../api/client'
import {
  resolveFilteredCombos,
  resolveFilteredGroups,
  resolveModelPickerGroups,
} from './pickerData'

describe('pickerData', () => {
  it('includes active connections and no-auth providers, excludes inactive connections', () => {
    const connections: ProviderConnection[] = [
      {
        id: 'conn-1',
        provider: 'antigravity',
        authType: 'oauth',
        name: 'My Antigravity',
        email: null,
        priority: 1,
        isActive: 1,
        data: '{}',
        createdAt: '',
        updatedAt: '',
      },
      {
        id: 'conn-2',
        provider: 'claude',
        authType: 'oauth',
        name: 'My Claude',
        email: null,
        priority: 1,
        isActive: 0,
        data: '{}',
        createdAt: '',
        updatedAt: '',
      },
    ]

    const groups = resolveModelPickerGroups(connections, [])

    const agGroup = groups.find((g) => g.id === 'antigravity')
    assert.ok(agGroup)
    assert.ok(agGroup.models.length > 0)
    assert.strictEqual(agGroup.models[0].value.startsWith('ag/'), true)

    const ccGroup = groups.find((g) => g.id === 'claude')
    assert.strictEqual(ccGroup, undefined)

    const ocGroup = groups.find((g) => g.id === 'opencode')
    assert.ok(ocGroup)
    assert.ok(ocGroup.models.length > 0)
  })

  it('excludes pure media providers and non-llm models from model picker groups', () => {
    const connections: ProviderConnection[] = [
      {
        id: 'c-el',
        provider: 'elevenlabs',
        authType: 'apikey',
        name: null,
        email: null,
        priority: null,
        isActive: 1,
        data: '{}',
        createdAt: '',
        updatedAt: '',
      },
      {
        id: 'c-oai',
        provider: 'openai',
        authType: 'apikey',
        name: null,
        email: null,
        priority: null,
        isActive: 1,
        data: '{}',
        createdAt: '',
        updatedAt: '',
      },
    ]

    const groups = resolveModelPickerGroups(connections, [])
    const elGroup = groups.find((g) => g.id === 'elevenlabs')
    assert.strictEqual(elGroup, undefined)

    const oaiGroup = groups.find((g) => g.id === 'openai')
    assert.ok(oaiGroup)
    assert.ok(oaiGroup.models.some((m) => m.id.includes('gpt')))
    assert.strictEqual(oaiGroup.models.some((m) => m.id === 'dall-e-3'), false)
    assert.strictEqual(oaiGroup.models.some((m) => m.id === 'tts-1'), false)
    assert.strictEqual(oaiGroup.models.some((m) => m.id === 'whisper-1'), false)
    assert.strictEqual(oaiGroup.models.some((m) => m.id === 'text-embedding-3-small'), false)
  })

  it('includes compatible nodes from providerNodes', () => {
    const nodes = [
      {
        id: 'custom-node-1',
        name: 'Custom OpenAI Node',
        type: 'openai-compatible',
        models: ['gpt-4o', 'gpt-4o-mini'],
      } as unknown as ProviderNode,
    ]

    const groups = resolveModelPickerGroups([], nodes)
    const customGroup = groups.find((g) => g.id === 'custom-node-1')
    assert.ok(customGroup)
    assert.strictEqual(customGroup.name, 'Custom OpenAI Node')
    assert.strictEqual(customGroup.models.length, 2)
    assert.strictEqual(customGroup.models[0].value, 'custom-node-1/gpt-4o')
    assert.strictEqual(customGroup.models[0].caps.vision, true)
  })

  it('filters combos correctly', () => {
    const combos: Combo[] = [
      {
        id: 'c1',
        name: 'combo-fast',
        kind: null,
        models: JSON.stringify(['ag/gemini-flash']),
        strategy: 'fallback',
        createdAt: '',
        updatedAt: '',
      },
      {
        id: 'c2',
        name: 'combo-smart',
        kind: null,
        models: JSON.stringify(['ag/gemini-pro']),
        strategy: 'fallback',
        createdAt: '',
        updatedAt: '',
      },
    ]

    assert.strictEqual(resolveFilteredCombos(combos, undefined, '', 'combo').length, 2)

    const withoutSelf = resolveFilteredCombos(combos, 'combo-fast', '', 'combo')
    assert.strictEqual(withoutSelf.length, 1)
    assert.strictEqual(withoutSelf[0].name, 'combo-smart')

    const searched = resolveFilteredCombos(combos, undefined, 'smart', 'combo')
    assert.strictEqual(searched.length, 1)
    assert.strictEqual(searched[0].name, 'combo-smart')

    assert.strictEqual(resolveFilteredCombos(combos, undefined, '', 'vision').length, 0)
    assert.strictEqual(resolveFilteredCombos(combos, undefined, '', 'audio').length, 0)
  })

  it('filters groups by search query and capability', () => {
    const groups = resolveModelPickerGroups(
      [
        {
          id: 'conn-1',
          provider: 'antigravity',
          authType: 'oauth',
          name: 'AG',
          email: null,
          priority: 1,
          isActive: 1,
          data: '{}',
          createdAt: '',
          updatedAt: '',
        },
      ],
      []
    )

    const flashGroups = resolveFilteredGroups(groups, 'flash', 'combo')
    assert.ok(flashGroups.length > 0)
    assert.strictEqual(
      flashGroups.every((g) =>
        g.models.every((m) => m.name.toLowerCase().includes('flash') || m.id.toLowerCase().includes('flash'))
      ),
      true
    )

    const visionGroups = resolveFilteredGroups(groups, '', 'vision')
    assert.strictEqual(
      visionGroups.every((g) => g.models.every((m) => m.caps.vision === true)),
      true
    )
  })
})
