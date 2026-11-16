import { z } from 'zod'
import type { Role } from '../../types'

export const roleOptions: Role[] = ['DEPARTMENT_USER', 'SUPERVISOR', 'WAREHOUSE_ADMIN']

export const baseSchema = z.object({
  email: z.string().email('validation.email'),
  full_name: z.string().min(2, 'validation.nameMin').max(200, 'validation.tooLong'),
  password: z.string().optional(),
  role: z.enum(['DEPARTMENT_USER', 'SUPERVISOR', 'WAREHOUSE_ADMIN'], {
    errorMap: () => ({ message: 'validation.selectRole' }),
  }),
  department_id: z.string().optional(),
})

export const userFormSchema = baseSchema.superRefine((values, ctx) => {
  if (values.role !== 'DEPARTMENT_USER') return
  const departmentID = values.department_id?.trim()
  if (!departmentID) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, path: ['department_id'], message: 'validation.departmentRequired' })
    return
  }
  if (!z.string().uuid().safeParse(departmentID).success) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, path: ['department_id'], message: 'validation.selectDepartment' })
  }
})

export type UserFormValues = z.infer<typeof baseSchema>
