import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, expect, it, vi } from 'vitest'
import { getIntelligenceTests, updateIntelligenceTests } from '@/api/admin/intelligenceTests'
import IntelligenceTestView from '../IntelligenceTestView.vue'

vi.mock('@/components/layout/AppLayout.vue', () => ({ default: { template: '<main><slot /></main>' } }))
vi.mock('@/components/icons/Icon.vue', () => ({ default: { template: '<span />' } }))
vi.mock('@/api/admin/intelligenceTests', () => ({
  getIntelligenceTests: vi.fn().mockResolvedValue({
    prompt: 'SVG', frequency_mins: 0, timeout_seconds: 120,
    groups: [{ id: 47, name: 'Test group', platform: 'openai', test_model: '',
      test_reasoning_effort: '', available_test_models: null, available_reasoning_efforts: null }],
    results: {}
  }),
  runIntelligenceTest: vi.fn(), updateIntelligenceTests: vi.fn()
}))

beforeEach(() => { vi.clearAllMocks() })

it('renders the page when an empty group returns null option lists', async () => {
  const wrapper = mount(IntelligenceTestView)
  await flushPromises()
  expect(wrapper.find('article').text()).toContain('Test group')
  const selects = wrapper.find('article').findAll('select')
  expect(selects).toHaveLength(2)
  expect(selects.every(select => select.attributes('disabled') !== undefined)).toBe(true)
  wrapper.unmount()
})

it('shows the normalized API error and retains the unsaved prompt', async () => {
  vi.mocked(updateIntelligenceTests).mockRejectedValueOnce({ status: 400, message: '选择的代理不存在或已停用' })
  const wrapper = mount(IntelligenceTestView)
  await flushPromises()
  await wrapper.get('textarea').setValue('new prompt')
  await wrapper.get('button.btn-primary').trigger('click')
  await flushPromises()
  expect(wrapper.text()).toContain('选择的代理不存在或已停用')
  expect(wrapper.get('textarea').element.value).toBe('new prompt')
  expect(wrapper.get('button.btn-primary').attributes('disabled')).toBeUndefined()
  wrapper.unmount()
})

it('saves long Chinese prompts without resubmitting every group model', async () => {
  const state = await getIntelligenceTests()
  vi.mocked(updateIntelligenceTests).mockResolvedValueOnce({ ...state, group_configs: {} })
  const wrapper = mount(IntelligenceTestView)
  await flushPromises()
  await wrapper.get('textarea').setValue('中'.repeat(4000))
  await wrapper.get('button.btn-primary').trigger('click')
  await flushPromises()
  expect(updateIntelligenceTests).toHaveBeenCalledTimes(1)
  const payload = vi.mocked(updateIntelligenceTests).mock.calls[0]![0]
  expect(payload.prompt).toHaveLength(4000)
  expect(payload).not.toHaveProperty('group_configs')
  wrapper.unmount()
})

it('rejects prompts over the character limit before submitting', async () => {
  const wrapper = mount(IntelligenceTestView)
  await flushPromises()
  await wrapper.get('textarea').setValue('中'.repeat(4001))
  await wrapper.get('button.btn-primary').trigger('click')
  await flushPromises()
  expect(updateIntelligenceTests).not.toHaveBeenCalled()
  expect(wrapper.text()).toContain('测试提示词最多支持 4000 个字符')
  wrapper.unmount()
})

it('submits only the group whose settings are saved', async () => {
  const state = await getIntelligenceTests()
  vi.mocked(getIntelligenceTests).mockResolvedValueOnce({ ...state, groups: [state.groups[0]!, { ...state.groups[0]!, id: 48, name: 'Offline' }] })
  vi.mocked(updateIntelligenceTests).mockResolvedValueOnce({ ...state, group_configs: {} })
  const wrapper = mount(IntelligenceTestView)
  await flushPromises()
  await wrapper.findAll('article')[0]!.findAll('button')[0]!.trigger('click')
  await flushPromises()
  expect(vi.mocked(updateIntelligenceTests).mock.calls[0]![0].group_configs).toEqual({ '47': { model: '', reasoning_effort: '' } })
  wrapper.unmount()
})

it('keeps an explicitly empty public group selection after loading and saving', async () => {
  const state = { ...await getIntelligenceTests(), public_group_filter_enabled: true, public_group_ids: [] }
  vi.mocked(getIntelligenceTests).mockResolvedValueOnce(state)
  vi.mocked(updateIntelligenceTests).mockResolvedValueOnce({ ...state, group_configs: {} })
  const wrapper = mount(IntelligenceTestView)
  await flushPromises()
  const boxes = wrapper.findAll<HTMLInputElement>('input[type="checkbox"]')
  expect(boxes[0]!.element.checked).toBe(true)
  expect(boxes[1]!.element.checked).toBe(false)
  await wrapper.get('button.btn-primary').trigger('click')
  await flushPromises()
  expect(vi.mocked(updateIntelligenceTests).mock.calls[0]![0].public_group_ids).toEqual([])
  wrapper.unmount()
})
