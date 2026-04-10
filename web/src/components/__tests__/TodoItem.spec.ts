import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import TodoItem from '../TodoItem.vue'

const baseTodo = { id: 1, title: 'Test todo', completed: false }

describe('TodoItem', () => {
  it('renders the todo title', () => {
    const wrapper = mount(TodoItem, { props: { todo: baseTodo } })
    expect(wrapper.text()).toContain('Test todo')
  })

  it('emits toggle when checkbox changes', async () => {
    const wrapper = mount(TodoItem, { props: { todo: baseTodo } })
    await wrapper.find('input[type="checkbox"]').trigger('change')
    expect(wrapper.emitted('toggle')).toEqual([[baseTodo]])
  })

  it('enters edit mode when edit button is clicked', async () => {
    const wrapper = mount(TodoItem, { props: { todo: baseTodo } })
    const buttons = wrapper.findAll('button')
    const editButton = buttons.find((b) => b.attributes('aria-label')?.startsWith('Edit'))!
    await editButton.trigger('click')
    expect(wrapper.find('input[aria-label="Edit todo title"]').exists()).toBe(true)
  })

  it('emits remove when delete button is clicked', async () => {
    const wrapper = mount(TodoItem, { props: { todo: baseTodo } })
    const buttons = wrapper.findAll('button')
    const deleteButton = buttons.find((b) => b.attributes('aria-label')?.startsWith('Delete'))!
    await deleteButton.trigger('click')
    expect(wrapper.emitted('remove')).toEqual([[1]])
  })

  it('applies done class when completed', () => {
    const wrapper = mount(TodoItem, {
      props: { todo: { ...baseTodo, completed: true } },
    })
    expect(wrapper.find('li').classes()).toContain('item--done')
  })
})
