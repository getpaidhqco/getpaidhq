import { ImageIcon } from 'lucide-react'
import { twMerge } from 'tailwind-merge'
import type {ProductResponse} from "@getpaidhq/sdk";

export const ProductThumbnail = ({
                                   size = 'small',
                                   product,
                                 }: {
  size?: 'small' | 'medium'
  product: ProductResponse
}) => {
  const coverUrl = null
  // todo once we have media support in the product schema, uncomment this
  // if (product.medias.length > 0) {
  //   coverUrl = product.medias[0].public_url
  // }

  const sizeClassName = size === 'small' ? 'h-10 rounded-md' : 'h-24 rounded-xl'

  return (
    <div
      className={twMerge(
        'dark:bg-gphq-800 dark:border-gphq-700 hidden aspect-square h-10 flex-col items-center justify-center border border-transparent bg-gray-100 text-center md:flex',
        sizeClassName,
      )}
    >
      {coverUrl ? (
        // eslint-disable-next-line @next/next/no-img-element
        <img
          src={coverUrl}
          alt={product.name}
          className={twMerge('aspect-square h-10 object-cover', sizeClassName)}
        />
      ) : (
        <ImageIcon
          className="size-6 text-muted-foreground"
        />
      )}
    </div>
  )
}
