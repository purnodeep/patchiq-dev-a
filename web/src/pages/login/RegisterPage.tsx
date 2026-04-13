import { useState, forwardRef } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useNavigate, useSearchParams, Link } from 'react-router';
import { Eye, EyeOff, UserPlus, AlertCircle, Loader2 } from 'lucide-react';
import { Skeleton } from '@patchiq/ui';
import { AuthLayout } from '../../components/auth/AuthLayout';
import { useValidateInvite, useRegister } from '../../api/hooks/useLogin';

const registerSchema = z
  .object({
    name: z.string().min(1, 'Full name is required'),
    password: z.string().min(8, 'Password must be at least 8 characters'),
    confirm_password: z.string().min(1, 'Please confirm your password'),
  })
  .refine((data) => data.password === data.confirm_password, {
    message: "Passwords don't match",
    path: ['confirm_password'],
  });

type RegisterFormValues = z.infer<typeof registerSchema>;

const inputStyle: React.CSSProperties = {
  width: '100%',
  height: '42px',
  padding: '0 12px',
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: '6px',
  fontSize: '13px',
  color: 'var(--text-primary)',
  fontFamily: 'var(--font-sans)',
  outline: 'none',
  transition: 'border-color 0.15s, box-shadow 0.15s',
  boxSizing: 'border-box',
};

const inputFocusStyle: React.CSSProperties = {
  borderColor: 'var(--accent)',
  boxShadow: '0 0 0 2px color-mix(in srgb, var(--accent) 15%, transparent)',
};

const labelStyle: React.CSSProperties = {
  display: 'block',
  fontSize: '12px',
  fontWeight: 500,
  color: 'var(--text-secondary)',
  marginBottom: '6px',
  fontFamily: 'var(--font-sans)',
};

const errorStyle: React.CSSProperties = {
  fontSize: '11px',
  color: 'var(--signal-critical)',
  marginTop: '4px',
  fontFamily: 'var(--font-sans)',
};

const FocusInput = forwardRef<HTMLInputElement, React.InputHTMLAttributes<HTMLInputElement>>(
  function FocusInput({ style, disabled, ...rest }, ref) {
    const [focused, setFocused] = useState(false);
    return (
      <input
        ref={ref}
        disabled={disabled}
        style={{
          ...inputStyle,
          ...(focused && !disabled ? inputFocusStyle : {}),
          ...(disabled
            ? { opacity: 0.5, cursor: 'not-allowed', background: 'var(--bg-inset)' }
            : {}),
          ...style,
        }}
        onFocus={() => setFocused(true)}
        onBlur={(e) => {
          setFocused(false);
          rest.onBlur?.(e);
        }}
        {...rest}
      />
    );
  },
);

export function RegisterPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const code = searchParams.get('code');

  const invite = useValidateInvite(code);
  const registerMutation = useRegister();

  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<RegisterFormValues>({
    resolver: zodResolver(registerSchema),
    defaultValues: {
      name: '',
      password: '',
      confirm_password: '',
    },
  });

  const onSubmit = (data: RegisterFormValues) => {
    if (!code) return;
    registerMutation.mutate(
      { code, name: data.name, password: data.password },
      {
        onSuccess: () => {
          void navigate('/');
        },
      },
    );
  };

  const cardStyle: React.CSSProperties = {
    background: 'var(--bg-card)',
    border: '1px solid var(--border)',
    borderRadius: '12px',
    padding: '32px',
    boxShadow: 'var(--shadow-sm)',
  };

  // No invite code in URL
  if (!code) {
    return (
      <AuthLayout>
        <div style={{ ...cardStyle, textAlign: 'center' }}>
          <div
            style={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              gap: '16px',
            }}
          >
            <div
              style={{
                display: 'flex',
                height: '52px',
                width: '52px',
                alignItems: 'center',
                justifyContent: 'center',
                borderRadius: '50%',
                background: 'color-mix(in srgb, var(--signal-critical) 8%, transparent)',
                border: '1px solid color-mix(in srgb, var(--signal-critical) 25%, transparent)',
              }}
            >
              <AlertCircle
                style={{ width: '24px', height: '24px', color: 'var(--signal-critical)' }}
              />
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
              <h1
                style={{
                  fontSize: '20px',
                  fontWeight: 700,
                  color: 'var(--text-emphasis)',
                  fontFamily: 'var(--font-sans)',
                }}
              >
                Invalid invite link
              </h1>
              <p
                style={{
                  fontSize: '13px',
                  color: 'var(--text-secondary)',
                  fontFamily: 'var(--font-sans)',
                }}
              >
                This invite link is invalid or has expired.
              </p>
            </div>
            <Link
              to="/login"
              style={{
                fontSize: '13px',
                color: 'var(--accent)',
                textDecoration: 'none',
                fontFamily: 'var(--font-sans)',
              }}
            >
              Back to sign in
            </Link>
          </div>
        </div>
      </AuthLayout>
    );
  }

  // Loading invite validation
  if (invite.isLoading) {
    return (
      <AuthLayout>
        <div style={cardStyle}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            <div style={{ textAlign: 'center' }}>
              <Skeleton className="h-8 w-48 mx-auto mb-2" />
              <Skeleton className="h-4 w-64 mx-auto" />
            </div>
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        </div>
      </AuthLayout>
    );
  }

  // Invalid or expired invite
  if (invite.isError) {
    return (
      <AuthLayout>
        <div style={{ ...cardStyle, textAlign: 'center' }}>
          <div
            style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '16px' }}
          >
            <div
              style={{
                display: 'flex',
                height: '52px',
                width: '52px',
                alignItems: 'center',
                justifyContent: 'center',
                borderRadius: '50%',
                background: 'color-mix(in srgb, var(--signal-critical) 8%, transparent)',
                border: '1px solid color-mix(in srgb, var(--signal-critical) 25%, transparent)',
              }}
            >
              <AlertCircle
                style={{ width: '24px', height: '24px', color: 'var(--signal-critical)' }}
              />
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
              <h1
                style={{
                  fontSize: '20px',
                  fontWeight: 700,
                  color: 'var(--text-emphasis)',
                  fontFamily: 'var(--font-sans)',
                }}
              >
                Invalid invite
              </h1>
              <p
                style={{
                  fontSize: '13px',
                  color: 'var(--text-secondary)',
                  fontFamily: 'var(--font-sans)',
                }}
              >
                {invite.error.message}
              </p>
            </div>
            <Link
              to="/login"
              style={{
                fontSize: '13px',
                color: 'var(--accent)',
                textDecoration: 'none',
                fontFamily: 'var(--font-sans)',
              }}
            >
              Back to sign in
            </Link>
          </div>
        </div>
      </AuthLayout>
    );
  }

  return (
    <AuthLayout>
      <div style={cardStyle}>
        <div style={{ textAlign: 'center', marginBottom: '28px' }}>
          <h1
            style={{
              fontSize: '22px',
              fontWeight: 700,
              color: 'var(--text-emphasis)',
              letterSpacing: '-0.02em',
              fontFamily: 'var(--font-sans)',
              marginBottom: '6px',
            }}
          >
            Create your account
          </h1>
          <p
            style={{
              fontSize: '13px',
              color: 'var(--text-secondary)',
              fontFamily: 'var(--font-sans)',
            }}
          >
            You've been invited to join{' '}
            <span style={{ color: 'var(--text-emphasis)', fontWeight: 500 }}>
              {invite.data?.tenant_name}
            </span>
            {invite.data?.role_name && (
              <>
                {' '}
                as{' '}
                <span style={{ color: 'var(--text-emphasis)', fontWeight: 500 }}>
                  {invite.data.role_name}
                </span>
              </>
            )}
          </p>
        </div>

        <form
          onSubmit={handleSubmit(onSubmit)}
          noValidate
          style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}
        >
          {/* Server error */}
          {registerMutation.isError && (
            <div
              style={{
                borderRadius: '6px',
                background: 'color-mix(in srgb, var(--signal-critical) 8%, transparent)',
                border: '1px solid color-mix(in srgb, var(--signal-critical) 25%, transparent)',
                padding: '10px 12px',
                fontSize: '13px',
                color: 'var(--signal-critical)',
                fontFamily: 'var(--font-sans)',
              }}
            >
              {registerMutation.error.message}
            </div>
          )}

          {/* Full name */}
          <div>
            <label htmlFor="name" style={labelStyle}>
              Full name
            </label>
            <FocusInput
              id="name"
              type="text"
              placeholder="Jane Doe"
              autoComplete="name"
              aria-invalid={!!errors.name}
              {...register('name')}
            />
            {errors.name && <p style={errorStyle}>{errors.name.message}</p>}
          </div>

          {/* Email (readonly from invite) */}
          <div>
            <label htmlFor="register-email" style={labelStyle}>
              Email
            </label>
            <FocusInput
              id="register-email"
              type="email"
              value={invite.data?.email ?? ''}
              disabled
            />
          </div>

          {/* Password */}
          <div>
            <label htmlFor="register-password" style={labelStyle}>
              Password
            </label>
            <div style={{ position: 'relative' }}>
              <FocusInput
                id="register-password"
                type={showPassword ? 'text' : 'password'}
                placeholder="At least 8 characters"
                autoComplete="new-password"
                aria-invalid={!!errors.password}
                style={{ paddingRight: '42px' }}
                {...register('password')}
              />
              <button
                type="button"
                style={{
                  position: 'absolute',
                  right: '12px',
                  top: '50%',
                  transform: 'translateY(-50%)',
                  background: 'none',
                  border: 'none',
                  cursor: 'pointer',
                  padding: 0,
                  display: 'flex',
                  color: 'var(--text-muted)',
                }}
                onClick={() => setShowPassword(!showPassword)}
                aria-label={showPassword ? 'Hide password' : 'Show password'}
              >
                {showPassword ? (
                  <EyeOff style={{ width: '16px', height: '16px' }} />
                ) : (
                  <Eye style={{ width: '16px', height: '16px' }} />
                )}
              </button>
            </div>
            {errors.password && <p style={errorStyle}>{errors.password.message}</p>}
          </div>

          {/* Confirm password */}
          <div>
            <label htmlFor="confirm-password" style={labelStyle}>
              Confirm password
            </label>
            <div style={{ position: 'relative' }}>
              <FocusInput
                id="confirm-password"
                type={showConfirmPassword ? 'text' : 'password'}
                placeholder="Re-enter your password"
                autoComplete="new-password"
                aria-invalid={!!errors.confirm_password}
                style={{ paddingRight: '42px' }}
                {...register('confirm_password')}
              />
              <button
                type="button"
                style={{
                  position: 'absolute',
                  right: '12px',
                  top: '50%',
                  transform: 'translateY(-50%)',
                  background: 'none',
                  border: 'none',
                  cursor: 'pointer',
                  padding: 0,
                  display: 'flex',
                  color: 'var(--text-muted)',
                }}
                onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                aria-label={showConfirmPassword ? 'Hide confirm password' : 'Show confirm password'}
              >
                {showConfirmPassword ? (
                  <EyeOff style={{ width: '16px', height: '16px' }} />
                ) : (
                  <Eye style={{ width: '16px', height: '16px' }} />
                )}
              </button>
            </div>
            {errors.confirm_password && <p style={errorStyle}>{errors.confirm_password.message}</p>}
          </div>

          {/* Submit button */}
          <button
            type="submit"
            disabled={registerMutation.isPending}
            style={{
              width: '100%',
              height: '44px',
              background: registerMutation.isPending
                ? 'color-mix(in srgb, var(--accent) 60%, transparent)'
                : 'var(--accent)',
              border: 'none',
              borderRadius: '6px',
              fontSize: '14px',
              fontWeight: 600,
              color: 'var(--text-on-color, #fff)',
              cursor: registerMutation.isPending ? 'not-allowed' : 'pointer',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              gap: '8px',
              fontFamily: 'var(--font-sans)',
              marginTop: '4px',
            }}
          >
            {registerMutation.isPending ? (
              <>
                <Loader2 style={{ width: '16px', height: '16px' }} className="animate-spin" />
                Creating account...
              </>
            ) : (
              <>
                <UserPlus style={{ width: '15px', height: '15px' }} />
                Create account
              </>
            )}
          </button>
        </form>

        {/* Login link */}
        <p
          style={{
            textAlign: 'center',
            fontSize: '13px',
            color: 'var(--text-muted)',
            marginTop: '20px',
            fontFamily: 'var(--font-sans)',
          }}
        >
          Already have an account?{' '}
          <Link
            to="/login"
            style={{
              color: 'var(--accent)',
              textDecoration: 'none',
              fontWeight: 500,
            }}
          >
            Sign in
          </Link>
        </p>
      </div>
    </AuthLayout>
  );
}
