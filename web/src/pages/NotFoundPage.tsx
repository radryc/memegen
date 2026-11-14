import { Link } from 'react-router-dom'
import { EmptyState } from '../components/UI'
import { Sparkles } from '../components/Icons'

export default function NotFoundPage() {
  return (
    <div className="grid min-h-[60vh] place-items-center">
      <div className="w-full max-w-lg">
        <EmptyState
          title="404 — this meme does not exist"
          description="The link may be broken, or the meme was too powerful for this dimension."
          action={
            <Link to="/" className="btn-primary">
              <Sparkles className="size-4" />
              Back to the board
            </Link>
          }
        />
      </div>
    </div>
  )
}
