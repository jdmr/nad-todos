import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import TodoForm from '../TodoForm.vue'

describe('TodoForm', () => {
  it('emits add event with trimmed title on submit', async () => {
    const wrapper = mount(TodoForm)
    const input = wrapper.find('input[type="text"]')
    await input.setValue('  Buy milk  ')
    await wrapper.find('form').trigger('submit')
    expect(wrapper.emitted('add')).toEqual([['Buy milk']])
    expect((input.element as HTMLInputElement).value).toBe('')
  })

  it('does not emit add event when title is empty', async () => {
    const wrapper = mount(TodoForm)
    await wrapper.find('input[type="text"]').setValue('   ')
    await wrapper.find('form').trigger('submit')
    expect(wrapper.emitted('add')).toBeUndefined()
  })

  it('disables the button when input is empty', () => {
    const wrapper = mount(TodoForm)
    const button = wrapper.find('button')
    expect(button.attributes('disabled')).toBeDefined()
  })
})
