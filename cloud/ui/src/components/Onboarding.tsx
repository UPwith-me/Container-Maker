import { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { useNavigate } from 'react-router-dom'
import {
    Rocket,
    Cloud,
    Server,
    Key,
    CheckCircle,
    ArrowRight,
    Sparkles
} from 'lucide-react'

interface OnboardingStep {
    id: string
    title: string
    description: string
    icon: React.ReactNode
    action?: () => void
    actionLabel?: string
    skippable?: boolean
}

export default function Onboarding({ onComplete }: { onComplete: () => void }) {
    const navigate = useNavigate()
    const [currentStep, setCurrentStep] = useState(0)
    const [completedSteps, setCompletedSteps] = useState<string[]>([])

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
            action: () => navigate('/settings'),
            actionLabel: 'Configure Providers',
            skippable: true,
        },
        {
            id: 'instance',
            title: 'Create Your First Instance',
            description: 'Launch a development environment in seconds. Start with Docker for local development, or use any configured cloud provider.',
            icon: <Server className="h-12 w-12 text-purple-500" />,
            action: () => navigate('/instances/new'),
            actionLabel: 'Create Instance',
            skippable: true,
        },
        {
            id: 'apikey',
            title: 'Generate an API Key',
            description: 'Use API keys for programmatic access via CLI or CI/CD pipelines. Keep them secure!',
            icon: <Key className="h-12 w-12 text-amber-500" />,
            action: () => navigate('/settings'),
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

    const handleNext = () => {
        const step = steps[currentStep]
        setCompletedSteps(prev => [...prev, step.id])

        if (currentStep === steps.length - 1) {
            localStorage.setItem('onboarding_completed', 'true')
            onComplete()
        } else {
            setCurrentStep(prev => prev + 1)
        }
    }

    const handleSkip = () => {
        handleNext()
    }

    const handleAction = () => {
        const step = steps[currentStep]
        if (step.action) {
            // Mark as completed and close onboarding
            setCompletedSteps(prev => [...prev, step.id])
            localStorage.setItem('onboarding_completed', 'true')
            step.action()
            onComplete()
        } else {
            handleNext()
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
                    className="w-full max-w-lg p-8 text-center"
                >
                    {/* Progress dots */}
                    <div className="flex justify-center gap-2 mb-8">
                        {steps.map((s, i) => (
                            <div
                                key={s.id}
                                className={`h-2 rounded-full transition-all ${i === currentStep
                                        ? 'w-8 bg-emerald-500'
                                        : i < currentStep
                                            ? 'w-2 bg-emerald-500/50'
                                            : 'w-2 bg-muted'
                                    }`}
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
