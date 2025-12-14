import { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { useNavigate } from 'react-router-dom'
import {
    Rocket,
    Cloud,
    Server,
    Key,
    ArrowRight,
    Sparkles,
    X
} from 'lucide-react'

interface OnboardingStep {
    id: string
    title: string
    description: string
    icon: React.ReactNode
    actionLabel: string
    navigateTo?: string  // Instead of action, just store the path
    skippable?: boolean
}

interface OnboardingProps {
    onComplete: () => void
    onMinimize?: () => void  // Allow minimizing instead of closing
}

export default function Onboarding({ onComplete, onMinimize }: OnboardingProps) {
    const navigate = useNavigate()
    const [currentStep, setCurrentStep] = useState(() => {
        // Resume from saved step
        const saved = localStorage.getItem('onboarding_step')
        return saved ? parseInt(saved, 10) : 0
    })

    const steps: OnboardingStep[] = [
        {
            id: 'welcome',
            title: 'Welcome to Container Maker! 🚀',
            description: 'Your unified cloud control plane for managing compute instances across multiple providers.',
            icon: <Sparkles className="h-12 w-12 text-emerald-500" />,
            actionLabel: 'Get Started',
        },
        {
            id: 'provider',
            title: 'Choose Your Cloud Provider',
            description: 'Docker local is ready to use! Configure additional cloud providers in Settings to unlock AWS, GCP, Azure, and more.',
            icon: <Cloud className="h-12 w-12 text-blue-500" />,
            navigateTo: '/settings',
            actionLabel: 'Configure Providers',
            skippable: true,
        },
        {
            id: 'instance',
            title: 'Create Your First Instance',
            description: 'Launch a development environment in seconds. Start with Docker for local development, or use any configured cloud provider.',
            icon: <Server className="h-12 w-12 text-purple-500" />,
            navigateTo: '/instances/new',
            actionLabel: 'Create Instance',
            skippable: true,
        },
        {
            id: 'apikey',
            title: 'Generate an API Key',
            description: 'Use API keys for programmatic access via CLI or CI/CD pipelines. Keep them secure!',
            icon: <Key className="h-12 w-12 text-amber-500" />,
            navigateTo: '/settings',
            actionLabel: 'Create API Key',
            skippable: true,
        },
        {
            id: 'done',
            title: 'You\'re All Set! 🎉',
            description: 'You now have everything you need to manage your cloud infrastructure. Happy building!',
            icon: <Rocket className="h-12 w-12 text-emerald-500" />,
            actionLabel: 'Go to Dashboard',
        },
    ]

    // Save current step to localStorage
    useEffect(() => {
        localStorage.setItem('onboarding_step', currentStep.toString())
    }, [currentStep])

    const handleNext = () => {
        if (currentStep === steps.length - 1) {
            // Only mark completed on final step
            localStorage.setItem('onboarding_completed', 'true')
            localStorage.removeItem('onboarding_step')
            onComplete()
        } else {
            setCurrentStep(prev => prev + 1)
        }
    }

    const handleAction = () => {
        const step = steps[currentStep]
        if (step.navigateTo) {
            // Save progress and navigate - DON'T mark as completed
            // Next time user opens onboarding, they continue from this step
            if (onMinimize) {
                onMinimize()
            }
            navigate(step.navigateTo)
        } else {
            handleNext()
        }
    }

    const handleSkip = () => {
        handleNext()
    }

    const handleClose = () => {
        // Just close without marking as completed - user can reopen anytime
        if (onMinimize) {
            onMinimize()
        } else {
            onComplete()
        }
    }

    const step = steps[currentStep]

    return (
        <AnimatePresence>
            <motion.div
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
                className="fixed inset-0 z-50 flex items-center justify-center bg-background/95 backdrop-blur-sm"
            >
                <motion.div
                    key={currentStep}
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    exit={{ opacity: 0, y: -20 }}
                    transition={{ duration: 0.3 }}
                    className="relative w-full max-w-lg p-8 text-center"
                >
                    {/* Close button */}
                    <button
                        onClick={handleClose}
                        className="absolute top-0 right-0 p-2 text-muted-foreground hover:text-foreground transition-colors"
                        title="Close (you can reopen from Settings)"
                    >
                        <X className="h-5 w-5" />
                    </button>

                    {/* Progress dots */}
                    <div className="flex justify-center gap-2 mb-8">
                        {steps.map((s, i) => (
                            <button
                                key={s.id}
                                onClick={() => setCurrentStep(i)}
                                className={`h-2 rounded-full transition-all cursor-pointer hover:opacity-80 ${i === currentStep
                                        ? 'w-8 bg-emerald-500'
                                        : i < currentStep
                                            ? 'w-2 bg-emerald-500/50'
                                            : 'w-2 bg-muted'
                                    }`}
                                title={`Step ${i + 1}: ${s.title}`}
                            />
                        ))}
                    </div>

                    {/* Icon */}
                    <div className="mb-6 flex justify-center">
                        <div className="p-4 rounded-2xl bg-muted/30">
                            {step.icon}
                        </div>
                    </div>

                    {/* Title */}
                    <h2 className="text-2xl font-bold mb-3">{step.title}</h2>

                    {/* Description */}
                    <p className="text-muted-foreground mb-8 text-lg">
                        {step.description}
                    </p>

                    {/* Actions */}
                    <div className="flex flex-col gap-3">
                        <button
                            onClick={handleAction}
                            className="w-full py-3 px-6 bg-emerald-500 hover:bg-emerald-600 text-white rounded-xl font-medium transition-colors flex items-center justify-center gap-2"
                        >
                            {step.actionLabel}
                            <ArrowRight className="h-4 w-4" />
                        </button>

                        {step.skippable && (
                            <button
                                onClick={handleSkip}
                                className="py-2 text-muted-foreground hover:text-foreground transition-colors"
                            >
                                Skip for now
                            </button>
                        )}
                    </div>

                    {/* Step counter */}
                    <p className="mt-6 text-sm text-muted-foreground">
                        Step {currentStep + 1} of {steps.length}
                    </p>
                </motion.div>
            </motion.div>
        </AnimatePresence>
    )
}

// Helper component to show "Restart Tour" button in settings or layout
export function RestartOnboardingButton({ onClick }: { onClick: () => void }) {
    return (
        <button
            onClick={() => {
                localStorage.removeItem('onboarding_completed')
                localStorage.setItem('onboarding_step', '0')
                onClick()
            }}
            className="text-sm text-emerald-500 hover:text-emerald-400 font-medium flex items-center gap-1"
        >
            <Sparkles className="h-4 w-4" />
            Restart Tour
        </button>
    )
}
