import { describe, expect, it } from 'vitest'
import { baseSchema, userFormSchema } from './userFormSchema'

const base = {
  email: 'name@example.com',
  full_name: 'Test User',
  password: 'Password123!',
  department_id: '',
}

describe('userFormSchema', () => {
  it('requires a department for department users', () => {
    const result = userFormSchema.safeParse({ ...base, role: 'DEPARTMENT_USER' })
    expect(result.success).toBe(false)
    expect(result.error?.issues[0]?.path).toEqual(['department_id'])
    expect(result.error?.issues[0]?.message).toBe('validation.departmentRequired')
  })

  it('rejects a malformed department id', () => {
    const result = userFormSchema.safeParse({ ...base, role: 'DEPARTMENT_USER', department_id: 'not-a-uuid' })
    expect(result.success).toBe(false)
    expect(result.error?.issues[0]?.message).toBe('validation.selectDepartment')
  })

  it('accepts the create default empty department for elevated roles', () => {
    expect(userFormSchema.safeParse({ ...base, role: 'SUPERVISOR' }).success).toBe(true)
    expect(userFormSchema.safeParse({ ...base, role: 'WAREHOUSE_ADMIN' }).success).toBe(true)
  })

  it('exposes role and department fields on the base schema for error mapping', () => {
    expect('role' in baseSchema.shape).toBe(true)
    expect('department_id' in baseSchema.shape).toBe(true)
  })

  it('rejects unknown roles', () => {
    const result = userFormSchema.safeParse({ ...base, role: 'OWNER' })
    expect(result.success).toBe(false)
    expect(result.error?.issues[0]?.message).toBe('validation.selectRole')
  })
})
