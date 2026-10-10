import { beforeEach, describe, expect, it, vi } from 'vitest'

import { apiClient } from '@/api/client'
import { updateSettings } from '@/api/admin/settings'

vi.mock('@/api/client', () => ({
  apiClient: {
    put: vi.fn(),
  },
}))

describe('admin settings update', () => {
  beforeEach(() => {
    vi.mocked(apiClient.put).mockReset()
  })

  it('allows enough time for the full settings payload to upload and save', async () => {
    vi.mocked(apiClient.put).mockResolvedValue({ data: { qq_group_enabled: true } })

    await updateSettings({ qq_group_enabled: true })

    expect(apiClient.put).toHaveBeenCalledWith(
      '/admin/settings',
      { qq_group_enabled: true },
      { timeout: 120000 },
    )
  })
})
